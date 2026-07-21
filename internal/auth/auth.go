package auth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("username/email atau password tidak sesuai")
	ErrSignupUnavailable  = errors.New("pendaftaran email tersedia setelah Supabase dikonfigurasi")
	ErrSessionIdle        = errors.New("sesi berakhir karena tidak ada aktivitas selama 30 menit")
)

const CookieName = "tpp_session"

const IdleTimeout = 30 * time.Minute

type Session struct {
	Subject        string   `json:"sub"`
	Email          string   `json:"email,omitempty"`
	DisplayName    string   `json:"name"`
	Role           string   `json:"role"`
	RoleID         string   `json:"role_id,omitempty"`
	RoleName       string   `json:"role_name,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
	SessionVersion int64    `json:"session_version,omitempty"`
	LastActivity   int64    `json:"last_activity"`
	ExpiresAt      int64    `json:"exp"`
}

func (s Session) Can(permission string) bool {
	if s.Role == "admin" {
		return true
	}
	for _, candidate := range s.Permissions {
		if candidate == permission {
			return true
		}
	}
	return false
}

func (s Session) CanAny(permissions ...string) bool {
	if s.Role == "admin" {
		return true
	}
	for _, permission := range permissions {
		if s.Can(permission) {
			return true
		}
	}
	return false
}

type SignupResult struct {
	UserID string
	Email  string
}

type VerifiedUser struct {
	UserID string
	Email  string
	Name   string
}

type Manager struct {
	secret          []byte
	secureCookie    bool
	adminUsername   string
	adminPassword   string
	supabaseURL     string
	supabaseAnonKey string
	publicBaseURL   string
	client          *http.Client
}

func NewManager(secret string, secureCookie bool, adminUsername, adminPassword, supabaseURL, supabaseAnonKey, publicBaseURL string) *Manager {
	return &Manager{
		secret: []byte(secret), secureCookie: secureCookie,
		adminUsername: adminUsername, adminPassword: adminPassword,
		supabaseURL: strings.TrimRight(supabaseURL, "/"), supabaseAnonKey: supabaseAnonKey,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"), client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *Manager) Login(ctx context.Context, identity, password string) (Session, error) {
	identity = strings.TrimSpace(identity)
	now := time.Now()
	if m.adminUsername != "" && m.adminPassword != "" && subtle.ConstantTimeCompare(hash(identity), hash(m.adminUsername)) == 1 && subtle.ConstantTimeCompare(hash(password), hash(m.adminPassword)) == 1 {
		return Session{Subject: "admin:" + m.adminUsername, DisplayName: "Administrator TPP", Role: "admin", RoleName: "Administrator", SessionVersion: m.localAdminSessionVersion(), LastActivity: now.Unix(), ExpiresAt: now.Add(8 * time.Hour).Unix()}, nil
	}
	if m.supabaseURL == "" || m.supabaseAnonKey == "" {
		return Session{}, ErrInvalidCredentials
	}
	payload := map[string]string{"email": identity, "password": password}
	var response struct {
		User struct {
			ID       string         `json:"id"`
			Email    string         `json:"email"`
			Metadata map[string]any `json:"user_metadata"`
		} `json:"user"`
	}
	if err := m.supabaseRequest(ctx, http.MethodPost, "/auth/v1/token?grant_type=password", payload, &response); err != nil {
		return Session{}, ErrInvalidCredentials
	}
	name := response.User.Email
	if value, ok := response.User.Metadata["name"].(string); ok && strings.TrimSpace(value) != "" {
		name = value
	}
	return Session{Subject: "user:" + response.User.ID, Email: response.User.Email, DisplayName: name, Role: "user", LastActivity: now.Unix(), ExpiresAt: now.Add(8 * time.Hour).Unix()}, nil
}

func hash(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func (m *Manager) localAdminSessionVersion() int64 {
	if m.adminUsername == "" || m.adminPassword == "" {
		return 0
	}
	sum := sha256.Sum256([]byte(m.adminUsername + "\x00" + m.adminPassword))
	// Keep the value positive and non-zero so a credential rotation invalidates
	// every previously signed local-admin session after the application restarts.
	version := int64(binary.BigEndian.Uint64(sum[:8]) & 0x7fffffffffffffff)
	if version == 0 {
		return 1
	}
	return version
}

func (m *Manager) ValidLocalAdminSession(session Session) bool {
	if session.Role != "admin" || m.adminUsername == "" || m.adminPassword == "" {
		return false
	}
	return subtle.ConstantTimeCompare(hash(session.Subject), hash("admin:"+m.adminUsername)) == 1 &&
		session.SessionVersion == m.localAdminSessionVersion()
}

func (m *Manager) Signup(ctx context.Context, name, email, password string) (SignupResult, error) {
	if m.supabaseURL == "" || m.supabaseAnonKey == "" {
		return SignupResult{}, ErrSignupUnavailable
	}
	email = strings.ToLower(strings.TrimSpace(email))
	payload := map[string]any{"email": email, "password": password, "data": map[string]string{"name": strings.TrimSpace(name)}}
	var response struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := m.supabaseRequest(ctx, http.MethodPost, "/auth/v1/signup", payload, &response); err != nil {
		return SignupResult{}, err
	}
	userID, returnedEmail := response.User.ID, response.User.Email
	if userID == "" {
		userID, returnedEmail = response.ID, response.Email
	}
	if userID == "" {
		return SignupResult{}, errors.New("pendaftaran tidak dapat dibuat atau email sudah pernah digunakan")
	}
	if returnedEmail == "" {
		returnedEmail = email
	}
	return SignupResult{UserID: userID, Email: returnedEmail}, nil
}

func (m *Manager) VerifySignupOTP(ctx context.Context, email, token string) (VerifiedUser, error) {
	if m.supabaseURL == "" || m.supabaseAnonKey == "" {
		return VerifiedUser{}, ErrSignupUnavailable
	}
	payload := map[string]string{"email": strings.ToLower(strings.TrimSpace(email)), "token": strings.TrimSpace(token), "type": "email"}
	var response struct {
		User struct {
			ID       string         `json:"id"`
			Email    string         `json:"email"`
			Metadata map[string]any `json:"user_metadata"`
		} `json:"user"`
	}
	if err := m.supabaseRequest(ctx, http.MethodPost, "/auth/v1/verify", payload, &response); err != nil {
		return VerifiedUser{}, err
	}
	if response.User.ID == "" {
		return VerifiedUser{}, errors.New("OTP tidak valid atau sudah kedaluwarsa")
	}
	name := response.User.Email
	if value, ok := response.User.Metadata["name"].(string); ok && strings.TrimSpace(value) != "" {
		name = strings.TrimSpace(value)
	}
	return VerifiedUser{UserID: response.User.ID, Email: response.User.Email, Name: name}, nil
}

func (m *Manager) ResendSignupOTP(ctx context.Context, email string) error {
	if m.supabaseURL == "" || m.supabaseAnonKey == "" {
		return ErrSignupUnavailable
	}
	payload := map[string]string{"email": strings.ToLower(strings.TrimSpace(email)), "type": "signup"}
	return m.supabaseRequest(ctx, http.MethodPost, "/auth/v1/resend", payload, nil)
}

func (m *Manager) RequestPasswordReset(ctx context.Context, email string) error {
	if m.supabaseURL == "" || m.supabaseAnonKey == "" {
		return ErrSignupUnavailable
	}
	payload := map[string]string{"email": strings.ToLower(strings.TrimSpace(email))}
	return m.supabaseRequest(ctx, http.MethodPost, "/auth/v1/recover", payload, nil)
}

// ResetPasswordWithOTP verifies a recovery OTP, immediately uses the returned
// short-lived user access token to update the password, and never exposes that
// token to the browser or stores it in an application session.
func (m *Manager) ResetPasswordWithOTP(ctx context.Context, email, token, password string) error {
	if m.supabaseURL == "" || m.supabaseAnonKey == "" {
		return ErrSignupUnavailable
	}
	payload := map[string]string{
		"email": strings.ToLower(strings.TrimSpace(email)),
		"token": strings.TrimSpace(token),
		"type":  "recovery",
	}
	var verification struct {
		AccessToken string `json:"access_token"`
	}
	if err := m.supabaseRequest(ctx, http.MethodPost, "/auth/v1/verify", payload, &verification); err != nil {
		return err
	}
	if strings.TrimSpace(verification.AccessToken) == "" {
		return errors.New("OTP pemulihan tidak menghasilkan sesi yang valid")
	}
	return m.supabaseRequestWithBearer(ctx, http.MethodPut, "/auth/v1/user", map[string]string{"password": password}, nil, verification.AccessToken)
}

func (m *Manager) supabaseRequest(ctx context.Context, method, path string, body any, out any) error {
	return m.supabaseRequestWithBearer(ctx, method, path, body, out, m.supabaseAnonKey)
}

func (m *Manager) supabaseRequestWithBearer(ctx context.Context, method, path string, body any, out any, bearer string) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, m.supabaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", m.supabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiError struct {
			Message          string `json:"msg"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.Unmarshal(data, &apiError)
		message := apiError.Message
		if message == "" {
			message = apiError.ErrorDescription
		}
		if message == "" {
			message = strings.TrimSpace(string(data))
		}
		return fmt.Errorf("supabase auth: %s", message)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (m *Manager) SetSession(w http.ResponseWriter, session Session) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	value := encoded + "." + m.sign(encoded)
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: value, Path: "/", HttpOnly: true, Secure: m.secureCookie, SameSite: http.SameSiteLaxMode, MaxAge: int(time.Until(time.Unix(session.ExpiresAt, 0)).Seconds())})
	return nil
}

func (m *Manager) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", HttpOnly: true, Secure: m.secureCookie, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}

func (m *Manager) Session(r *http.Request) (Session, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return Session{}, err
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 || !hmac.Equal([]byte(parts[1]), []byte(m.sign(parts[0]))) {
		return Session{}, ErrInvalidCredentials
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return Session{}, err
	}
	now := time.Now()
	if session.Subject == "" || session.ExpiresAt <= now.Unix() {
		return Session{}, ErrInvalidCredentials
	}
	if session.LastActivity <= 0 || now.Sub(time.Unix(session.LastActivity, 0)) >= IdleTimeout {
		return Session{}, ErrSessionIdle
	}
	return session, nil
}

// TouchSession refreshes the signed last-activity timestamp while preserving
// the absolute session expiry and the CSRF token derived from it.
func (m *Manager) TouchSession(w http.ResponseWriter, session Session) error {
	session.LastActivity = time.Now().Unix()
	return m.SetSession(w, session)
}

func (m *Manager) CSRFToken(session Session) string {
	return m.sign("csrf:" + session.Subject + ":" + fmt.Sprint(session.ExpiresAt))
}

func (m *Manager) ValidateCSRF(session Session, token string) bool {
	return hmac.Equal([]byte(m.CSRFToken(session)), []byte(token))
}

func (m *Manager) sign(value string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
