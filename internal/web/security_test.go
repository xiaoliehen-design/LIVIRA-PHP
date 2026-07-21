package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterEnforcesWindow(t *testing.T) {
	limiter := newRateLimiter()
	if !limiter.Allow("login:127.0.0.1:user", 2, time.Minute) {
		t.Fatal("first attempt should be allowed")
	}
	if !limiter.Allow("login:127.0.0.1:user", 2, time.Minute) {
		t.Fatal("second attempt should be allowed")
	}
	if limiter.Allow("login:127.0.0.1:user", 2, time.Minute) {
		t.Fatal("third attempt should be blocked")
	}
	limiter.Reset("login:127.0.0.1:user")
	if !limiter.Allow("login:127.0.0.1:user", 2, time.Minute) {
		t.Fatal("attempt should be allowed after reset")
	}
}

func TestSecurityHeadersAreApplied(t *testing.T) {
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	if recorder.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing frame protection: %q", recorder.Header().Get("X-Frame-Options"))
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing content type protection: %q", recorder.Header().Get("X-Content-Type-Options"))
	}
	if recorder.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing content security policy")
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("dynamic response must not be cached: %q", recorder.Header().Get("Cache-Control"))
	}
}
