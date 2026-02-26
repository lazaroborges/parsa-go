package config

import (
	"os"
	"testing"
)

func setRequiredEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-jwt-secret-key")
	t.Setenv("ENCRYPTION_KEY", "01234567890123456789012345678901") // 32 bytes
}

func TestLoad_Success(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.JWT.Secret != "test-jwt-secret-key" {
		t.Errorf("JWT.Secret = %q, want %q", cfg.JWT.Secret, "test-jwt-secret-key")
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "8080")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ENCRYPTION_KEY", "01234567890123456789012345678901")
	os.Unsetenv("JWT_SECRET")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for missing JWT_SECRET, got nil")
	}
}

func TestLoad_InvalidEncryptionKeyLength(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ENCRYPTION_KEY", "too-short")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid ENCRYPTION_KEY length, got nil")
	}
}

func TestLoad_MissingEncryptionKey(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ENCRYPTION_KEY", "")
	os.Unsetenv("ENCRYPTION_KEY")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for missing ENCRYPTION_KEY, got nil")
	}
}

func TestLoad_InvalidDBPort(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("DB_PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid DB_PORT, got nil")
	}
}

func TestLoad_TLSValidation(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("TLS_ENABLED", "true")
	t.Setenv("TLS_CERT_PATH", "")
	t.Setenv("TLS_KEY_PATH", "")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for TLS enabled without cert path, got nil")
	}
}

func TestLoad_TLSValidation_MissingKeyPath(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("TLS_ENABLED", "true")
	t.Setenv("TLS_CERT_PATH", "/path/to/cert")
	t.Setenv("TLS_KEY_PATH", "")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for TLS enabled without key path, got nil")
	}
}

func TestLoad_AllowedHosts(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("ALLOWED_HOSTS", "example.com, api.example.com, localhost:3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(cfg.Server.AllowedHosts) != 3 {
		t.Errorf("AllowedHosts length = %d, want 3", len(cfg.Server.AllowedHosts))
	}
}

func TestLoad_SchedulerConfig(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("SCHEDULER_ENABLED", "false")
	t.Setenv("SCHEDULER_WORKERS", "10")
	t.Setenv("SCHEDULER_RUN_ON_STARTUP", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Scheduler.Enabled != false {
		t.Error("Scheduler.Enabled should be false")
	}
	if cfg.Scheduler.WorkerCount != 10 {
		t.Errorf("Scheduler.WorkerCount = %d, want 10", cfg.Scheduler.WorkerCount)
	}
	if cfg.Scheduler.RunOnStartup != true {
		t.Error("Scheduler.RunOnStartup should be true")
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		value    string
		defVal   bool
		expected bool
	}{
		{"true", false, true},
		{"TRUE", false, true},
		{"True", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"YES", false, true},
		{"false", true, false},
		{"FALSE", true, false},
		{"0", true, false},
		{"no", true, false},
		{"NO", true, false},
		{"invalid", true, true},   // returns default
		{"invalid", false, false}, // returns default
		{"", true, true},          // empty returns default
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			key := "TEST_BOOL_ENV"
			if tt.value == "" {
				os.Unsetenv(key)
			} else {
				t.Setenv(key, tt.value)
			}

			got := getBoolEnv(key, tt.defVal)
			if got != tt.expected {
				t.Errorf("getBoolEnv(%q, %v) = %v, want %v", tt.value, tt.defVal, got, tt.expected)
			}
		})
	}
}

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	got := cfg.ConnectionString()
	if got != expected {
		t.Errorf("ConnectionString() = %q, want %q", got, expected)
	}
}

func TestLoad_OAuthCallbackURLs(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("HOST_URL", "https://api.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.OAuth.Google.WebCallbackURL != "https://api.example.com/api/auth/oauth/callback" {
		t.Errorf("Google WebCallbackURL = %q", cfg.OAuth.Google.WebCallbackURL)
	}
	if cfg.OAuth.Google.MobileCallbackURL != "https://api.example.com/api/auth/oauth/mobile/callback" {
		t.Errorf("Google MobileCallbackURL = %q", cfg.OAuth.Google.MobileCallbackURL)
	}
}
