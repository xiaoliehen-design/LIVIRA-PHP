package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hendra/manajemen-tpp/internal/auth"
)

func TestCaptchaChallengeVerifiesAndExpires(t *testing.T) {
	manager := newCaptchaManager("test-secret-long-enough")
	current := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return current }
	token, answer, err := manager.newChallenge()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || len(answer) != captchaLength {
		t.Fatalf("invalid challenge: token=%q answer=%q", token, answer)
	}
	if !manager.verify(token, strings.ToLower(answer)) {
		t.Fatal("valid CAPTCHA answer was rejected")
	}
	if manager.verify(token, "WRONG") {
		t.Fatal("invalid CAPTCHA answer was accepted")
	}
	if manager.verify(token+"x", answer) {
		t.Fatal("tampered CAPTCHA token was accepted")
	}
	current = current.Add(captchaTTL + time.Second)
	if manager.verify(token, answer) {
		t.Fatal("expired CAPTCHA token was accepted")
	}
}

func TestCaptchaRendersPNGWithoutCaching(t *testing.T) {
	handler := testHandler(t)
	manager := newCaptchaManager("test-secret-long-enough")
	token, _, err := manager.newChallenge()
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/captcha.png?token="+url.QueryEscape(token), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected CAPTCHA SVG 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Header().Get("Content-Type"), "image/png") || !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("CAPTCHA PNG headers are not secure")
	}
	if body := recorder.Body.Bytes(); len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("CAPTCHA PNG was not rendered")
	}
}

func TestLoginRejectsMissingCaptcha(t *testing.T) {
	form := url.Values{"identity": {"test-admin"}, "password": {"local-test-password-2026"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther || !strings.Contains(recorder.Header().Get("Location"), "CAPTCHA") {
		t.Fatalf("missing CAPTCHA must be rejected, got status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == auth.CookieName && cookie.Value != "" {
			t.Fatal("login without CAPTCHA created a session")
		}
	}
}
