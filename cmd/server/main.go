package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/hendra/manajemen-tpp/internal/auth"
	"github.com/hendra/manajemen-tpp/internal/config"
	"github.com/hendra/manajemen-tpp/internal/store"
	appweb "github.com/hendra/manajemen-tpp/internal/web"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if location, err := time.LoadLocation(env("TZ", "Asia/Jakarta")); err == nil {
		time.Local = location
	}
	if cfg.Production() {
		if err := validateProductionConfig(cfg); err != nil {
			logger.Error("konfigurasi production tidak aman", "error", err)
			os.Exit(1)
		}
	}

	var dataStore store.Store
	if cfg.DemoMode {
		dataStore = store.NewMemoryStore()
		logger.Info("application mode", "mode", "demo", "persistence", "memory")
	} else {
		if !cfg.SupabaseConfigured() {
			logger.Error("konfigurasi Supabase belum lengkap")
			os.Exit(1)
		}
		dataStore = store.NewSupabaseStore(cfg.SupabaseURL, cfg.SupabaseServiceKey, cfg.StorageBucket)
		logger.Info("application mode", "mode", "production", "persistence", "supabase")
	}

	authManager := auth.NewManager(cfg.SessionSecret, cfg.Production(), cfg.AdminUsername, cfg.AdminPassword, cfg.SupabaseURL, cfg.SupabaseAnonKey, cfg.PublicBaseURL)
	app, err := appweb.NewServer(cfg, dataStore, authManager, logger)
	if err != nil {
		logger.Error("initialize server", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 8 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	go func() {
		logger.Info("server started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
	logger.Info("server stopped")
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func validateProductionConfig(cfg config.Config) error {
	if cfg.DemoMode {
		return errors.New("DEMO_MODE harus false")
	}
	if len(cfg.SessionSecret) < 32 || cfg.SessionSecret == "change-this-session-secret-before-production" {
		return errors.New("SESSION_SECRET wajib acak dan minimal 32 karakter")
	}
	if !strings.HasPrefix(strings.ToLower(cfg.PublicBaseURL), "https://") {
		return errors.New("PUBLIC_BASE_URL wajib menggunakan HTTPS")
	}
	adminUsernameSet := strings.TrimSpace(cfg.AdminUsername) != ""
	adminPasswordSet := cfg.AdminPassword != ""
	if adminUsernameSet != adminPasswordSet {
		return errors.New("ADMIN_USERNAME dan ADMIN_PASSWORD harus sama-sama diisi atau sama-sama dikosongkan")
	}
	if adminUsernameSet {
		if len(cfg.AdminPassword) < 16 {
			return errors.New("ADMIN_PASSWORD minimal 16 karakter")
		}
		blocked := []string{"ganteng123456", "password", "admin123", "admin-demo-only"}
		for _, value := range blocked {
			if strings.EqualFold(cfg.AdminPassword, value) {
				return errors.New("ADMIN_PASSWORD masih berupa password umum/default")
			}
		}
	}
	if strings.TrimSpace(cfg.StorageBucket) == "" {
		return errors.New("SUPABASE_STORAGE_BUCKET wajib diisi")
	}
	return nil
}
