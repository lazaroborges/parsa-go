package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	OAuth      OAuthConfig
	JWT        JWTConfig
	Encryption EncryptionConfig
	Scheduler  SchedulerConfig
	TLS        TLSConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type OAuthConfig struct {
	Google GoogleOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type JWTConfig struct {
	Secret string
}

type EncryptionConfig struct {
	Key string
}

type SchedulerConfig struct {
	Enabled       bool
	ScheduleTimes []string
	WorkerCount   int
	JobDelay      time.Duration
	QueueSize     int
	RunOnStartup  bool
}

type TLSConfig struct {
	Enabled      bool
	CertPath     string
	KeyPath      string
	RedirectHTTP bool
}

// Load constructs a Config by reading environment variables, applying sensible defaults,
// and validating required values.
// It parses numeric and duration settings used by the database and scheduler, and reads
// TLS, OAuth, JWT, and encryption settings from the environment.
// Returns a pointer to the populated Config or an error if any environment value is invalid
// or a required value is missing (for example: missing JWT_SECRET, missing ENCRYPTION_KEY,
// ENCRYPTION_KEY must be exactly 32 bytes, and TLS_CERT_PATH/TLS_KEY_PATH are required when TLS_ENABLED=true).
func Load() (*Config, error) {

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	// Parse scheduler configuration
	schedulerEnabled := getBoolEnv("SCHEDULER_ENABLED", true)
	schedulerTimes := strings.Split(getEnv("SCHEDULER_TIMES", "05:00,10:00,14:00,20:00"), ",")
	schedulerWorkers, err := strconv.Atoi(getEnv("SCHEDULER_WORKERS", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid SCHEDULER_WORKERS: %w", err)
	}
	schedulerJobDelay, err := time.ParseDuration(getEnv("SCHEDULER_JOB_DELAY", "1s"))
	if err != nil {
		return nil, fmt.Errorf("invalid SCHEDULER_JOB_DELAY: %w", err)
	}
	schedulerQueueSize, err := strconv.Atoi(getEnv("SCHEDULER_QUEUE_SIZE", "100"))
	if err != nil {
		return nil, fmt.Errorf("invalid SCHEDULER_QUEUE_SIZE: %w", err)
	}
	schedulerRunOnStartup := getBoolEnv("SCHEDULER_RUN_ON_STARTUP", false)

	// Parse TLS configuration
	tlsEnabled := getBoolEnv("TLS_ENABLED", false)
	tlsCertPath := getEnv("TLS_CERT_PATH", "")
	tlsKeyPath := getEnv("TLS_KEY_PATH", "")
	tlsRedirectHTTP := getBoolEnv("TLS_REDIRECT_HTTP", false)

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "lazaro"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "parsa-go"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		OAuth: OAuthConfig{
			Google: GoogleOAuthConfig{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
			},
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
		},
		Encryption: EncryptionConfig{
			Key: getEnv("ENCRYPTION_KEY", ""),
		},
		Scheduler: SchedulerConfig{
			Enabled:       schedulerEnabled,
			ScheduleTimes: schedulerTimes,
			WorkerCount:   schedulerWorkers,
			JobDelay:      schedulerJobDelay,
			QueueSize:     schedulerQueueSize,
			RunOnStartup:  schedulerRunOnStartup,
		},
		TLS: TLSConfig{
			Enabled:      tlsEnabled,
			CertPath:     tlsCertPath,
			KeyPath:      tlsKeyPath,
			RedirectHTTP: tlsRedirectHTTP,
		},
	}

	// Validate required fields
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.Encryption.Key == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required")
	}
	if len(cfg.Encryption.Key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes for AES-256")
	}

	// Validate TLS configuration
	if cfg.TLS.Enabled {
		if cfg.TLS.CertPath == "" {
			return nil, fmt.Errorf("TLS_CERT_PATH is required when TLS_ENABLED=true")
		}
		if cfg.TLS.KeyPath == "" {
			return nil, fmt.Errorf("TLS_KEY_PATH is required when TLS_ENABLED=true")
		}
	}

	return cfg, nil
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Accept: true, false, 1, 0, yes, no (case-insensitive)
	switch strings.ToLower(value) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return defaultValue
	}
}