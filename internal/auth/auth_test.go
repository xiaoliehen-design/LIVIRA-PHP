package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionRejectsIdleCookie(t *testing.T) {
	manager := NewManager("test-secret", false, "admin", "password", "", "", "")
	recorder := httptest.NewRecorder()
	session := Session{
		Subject:      "admin:admin",
		DisplayName:  "Administrator",
		Role:         "admin",
		LastActivity: time.Now().Add(-IdleTimeout - time.Minute).Unix(),
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}
	if err := manager.SetSession(recorder, session); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(recorder.Result().Cookies()[0])
	if _, err := manager.Session(request); !errors.Is(err, ErrSessionIdle) {
		t.Fatalf("expected ErrSessionIdle, got %v", err)
	}
}

func TestTouchSessionRefreshesLastActivity(t *testing.T) {
	manager := NewManager("test-secret", false, "admin", "password", "", "", "")
	session := Session{
		Subject:      "admin:admin",
		DisplayName:  "Administrator",
		Role:         "admin",
		LastActivity: time.Now().Add(-10 * time.Minute).Unix(),
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}
	recorder := httptest.NewRecorder()
	if err := manager.TouchSession(recorder, session); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(recorder.Result().Cookies()[0])
	refreshed, err := manager.Session(request)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(time.Unix(refreshed.LastActivity, 0)) > 5*time.Second {
		t.Fatalf("last activity was not refreshed: %v", refreshed.LastActivity)
	}
}

func TestLocalAdminDisabledWithoutCredentials(t *testing.T) {
	manager := NewManager("test-secret-with-more-than-32-characters", false, "", "", "", "", "")
	if _, err := manager.Login(context.Background(), "admin", "anything"); err == nil {
		t.Fatal("local admin login must be disabled when credentials are empty")
	}
}

func TestLocalAdminCredentialRotationInvalidatesOldSession(t *testing.T) {
	oldManager := NewManager("test-secret-with-more-than-32-characters", false, "break-glass", "a-strong-admin-password-2026", "", "", "")
	session, err := oldManager.Login(context.Background(), "break-glass", "a-strong-admin-password-2026")
	if err != nil {
		t.Fatal(err)
	}
	if !oldManager.ValidLocalAdminSession(session) {
		t.Fatal("newly created local-admin session should be valid")
	}
	rotatedManager := NewManager("test-secret-with-more-than-32-characters", false, "break-glass", "a-different-admin-password-2026", "", "", "")
	if rotatedManager.ValidLocalAdminSession(session) {
		t.Fatal("session signed before an admin credential rotation must be rejected")
	}
}

func TestPasswordResetUsesRecoveryOTPAndShortLivedAccessToken(t *testing.T) {
	var requestedPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/v1/recover":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["email"] != "pegawai@example.go.id" {
				t.Fatalf("unexpected recovery request: body=%v err=%v", body, err)
			}
			_, _ = w.Write([]byte(`{}`))
		case "/auth/v1/verify":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["email"] != "pegawai@example.go.id" || body["token"] != "123456" || body["type"] != "recovery" {
				t.Fatalf("unexpected recovery verification: body=%v err=%v", body, err)
			}
			_, _ = w.Write([]byte(`{"access_token":"recovery-access-token"}`))
		case "/auth/v1/user":
			if r.Header.Get("Authorization") != "Bearer recovery-access-token" || r.Header.Get("apikey") != "anon-key" {
				t.Fatalf("password update used invalid authorization headers: %#v", r.Header)
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["password"] != "new-secure-password" {
				t.Fatalf("unexpected password update: body=%v err=%v", body, err)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	manager := NewManager("test-secret", false, "", "", server.URL, "anon-key", "")
	if err := manager.RequestPasswordReset(context.Background(), " PEGAWAI@EXAMPLE.GO.ID "); err != nil {
		t.Fatal(err)
	}
	if err := manager.ResetPasswordWithOTP(context.Background(), "pegawai@example.go.id", "123456", "new-secure-password"); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(requestedPaths, ",")
	for _, expected := range []string{"POST /auth/v1/recover", "POST /auth/v1/verify", "PUT /auth/v1/user"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("password reset did not call %s: %s", expected, joined)
		}
	}
}
