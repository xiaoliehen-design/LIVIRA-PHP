package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port               string
	AppName            string
	AppEnv             string
	PublicBaseURL      string
	SessionSecret      string
	AdminUsername      string
	AdminPassword      string
	SupabaseURL        string
	SupabaseAnonKey    string
	SupabaseServiceKey string
	StorageBucket      string
	DemoMode           bool
}

func Load() Config {
	cfg := Config{
		Port:               value("PORT", "8080"),
		AppName:            value("APP_NAME", "LIVIRA"),
		AppEnv:             value("APP_ENV", "development"),
		PublicBaseURL:      strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"),
		SessionSecret:      value("SESSION_SECRET", "change-this-session-secret-before-production"),
		AdminUsername:      strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		SupabaseURL:        strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		SupabaseAnonKey:    os.Getenv("SUPABASE_ANON_KEY"),
		SupabaseServiceKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		StorageBucket:      value("SUPABASE_STORAGE_BUCKET", "livira-documents"),
	}
	demoDefault := !cfg.SupabaseConfigured()
	cfg.DemoMode = boolValue("DEMO_MODE", demoDefault)
	if !cfg.Production() && cfg.DemoMode && cfg.AdminUsername == "" && cfg.AdminPassword == "" {
		cfg.AdminUsername = "admin"
		cfg.AdminPassword = "admin-demo-only"
	}
	return cfg
}

func (c Config) Production() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func (c Config) SupabaseConfigured() bool {
	return c.SupabaseURL != "" && c.SupabaseAnonKey != "" && c.SupabaseServiceKey != ""
}

func value(key, fallback string) string {
	if result := strings.TrimSpace(os.Getenv(key)); result != "" {
		return result
	}
	return fallback
}

func boolValue(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	result, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return result
}
