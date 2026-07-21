package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hendra/manajemen-tpp/internal/auth"
	"github.com/hendra/manajemen-tpp/internal/domain"
)

const maxRequestBodyBytes int64 = 12 << 20
const requestIDKey contextKey = "request_id"

type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	calls    int
}

func newRateLimiter() *rateLimiter { return &rateLimiter{attempts: make(map[string][]time.Time)} }
func (l *rateLimiter) Allow(key string, limit int, window time.Duration) bool {
	now, cutoff := time.Now(), time.Now().Add(-window)
	l.mu.Lock()
	defer l.mu.Unlock()
	values := l.attempts[key]
	kept := values[:0]
	for _, v := range values {
		if v.After(cutoff) {
			kept = append(kept, v)
		}
	}
	if len(kept) >= limit {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	l.calls++
	if l.calls%200 == 0 {
		for candidate, timestamps := range l.attempts {
			active := timestamps[:0]
			for _, t := range timestamps {
				if t.After(now.Add(-24 * time.Hour)) {
					active = append(active, t)
				}
			}
			if len(active) == 0 {
				delete(l.attempts, candidate)
			} else {
				l.attempts[candidate] = active
			}
		}
	}
	return true
}
func (l *rateLimiter) Reset(key string) { l.mu.Lock(); delete(l.attempts, key); l.mu.Unlock() }

type parameterCache struct {
	mu       sync.RWMutex
	options  []domain.ParameterOption
	loadedAt time.Time
}

func (s *Server) loadRuntimeParameters(ctx context.Context, force bool) error {
	s.parameters.mu.RLock()
	fresh := !force && len(s.parameters.options) > 0 && time.Since(s.parameters.loadedAt) < 5*time.Minute
	cached := append([]domain.ParameterOption(nil), s.parameters.options...)
	s.parameters.mu.RUnlock()
	if fresh {
		domain.SetRuntimeParameters(cached)
		return nil
	}
	options, err := s.store.ParameterOptions(ctx, "", true)
	if err != nil {
		if len(cached) > 0 {
			domain.SetRuntimeParameters(cached)
		}
		return err
	}
	s.parameters.mu.Lock()
	s.parameters.options = append([]domain.ParameterOption(nil), options...)
	s.parameters.loadedAt = time.Now()
	s.parameters.mu.Unlock()
	domain.SetRuntimeParameters(options)
	return nil
}
func (s *Server) invalidateParameterCache() {
	s.parameters.mu.Lock()
	s.parameters.loadedAt = time.Time{}
	s.parameters.mu.Unlock()
}
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
func (s *Server) authAttemptAllowed(r *http.Request, action, identity string, limit int, window time.Duration) bool {
	return s.authLimiter.Allow(action+":"+clientIP(r)+":"+strings.ToLower(strings.TrimSpace(identity)), limit, window)
}
func (s *Server) resetAuthAttempts(r *http.Request, action, identity string) {
	s.authLimiter.Reset(action + ":" + clientIP(r) + ":" + strings.ToLower(strings.TrimSpace(identity)))
}
func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
func requestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}
func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" || len(id) > 128 {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}
func (s *Server) limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) writeAudit(r *http.Request, action, entityType, entityID, outcome string, metadata map[string]any) {
	session, _ := sessionFromContext(r.Context())
	entry := domain.AuditEntry{ActorSubject: session.Subject, ActorName: session.DisplayName, Action: action, EntityType: entityType, EntityID: entityID, Outcome: outcome, IPAddress: clientIP(r), UserAgent: r.UserAgent(), RequestID: requestIDFromContext(r.Context()), Metadata: metadata}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.store.WriteAudit(ctx, entry); err != nil {
		s.logger.Warn("write audit", "action", action, "error", err)
	}
}
func (s *Server) refreshUserSession(ctx context.Context, session auth.Session) (auth.Session, error) {
	if session.Role == "admin" {
		if !s.auth.ValidLocalAdminSession(session) {
			return auth.Session{}, auth.ErrInvalidCredentials
		}
		return session, nil
	}
	id := strings.TrimPrefix(session.Subject, "user:")
	if id == "" || id == session.Subject {
		return auth.Session{}, auth.ErrInvalidCredentials
	}
	account, err := s.store.UserByAuthID(ctx, id)
	if err != nil {
		return auth.Session{}, err
	}
	if !account.EmailVerified || account.ApprovalStatus != "approved" || account.RoleID == "" || account.RoleName == "" || len(account.Permissions) == 0 || session.SessionVersion != account.SessionVersion {
		return auth.Session{}, auth.ErrInvalidCredentials
	}
	session.DisplayName, session.Email, session.RoleID, session.RoleName = account.Name, account.Email, account.RoleID, account.RoleName
	session.Permissions = append([]string(nil), account.Permissions...)
	return session, nil
}
func (s *Server) documentAllowed(ctx context.Context, session auth.Session, documentID string) (bool, error) {
	accesses, err := s.store.DocumentAccess(ctx, documentID)
	if err != nil {
		return false, err
	}
	for _, a := range accesses {
		if !sessionCanAccessItem(session, a.Inventory) {
			continue
		}
		if a.DispositionType != "" {
			view, _ := processPermissions(domain.DispositionType(a.DispositionType))
			if view != "" && session.Can(view) {
				return true, nil
			}
			continue
		}
		if strings.HasPrefix(a.EventCode, "rekonsiliasi") && session.Can(domain.PermissionReconciliationView) {
			return true, nil
		}
		if session.CanAny(domain.PermissionInventoryView, domain.PermissionSearchView, domain.PermissionReconciliationView) {
			return true, nil
		}
	}
	return false, nil
}
func isBodyTooLarge(err error) bool { var e *http.MaxBytesError; return errors.As(err, &e) }

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}
func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
