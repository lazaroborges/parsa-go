package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	OAuth       OAuthConfig
	JWT         JWTConfig
	Encryption  EncryptionConfig
	Scheduler   SchedulerConfig
	TLS         TLSConfig
	OpenFinance OpenFinanceConfig
	Firebase    FirebaseConfig
	Telemetry   TelemetryConfig
}

type ServerConfig struct {
	Port         string
	Host         string
	AllowedHosts []string
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
	Apple  AppleOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID          string
	ClientSecret      string
	WebCallbackURL    string
	MobileCallbackURL string
}

type AppleOAuthConfig struct {
	TeamID            string
	KeyID             string
	ClientID          string
	PrivateKeyPath    string
	MobileCallbackURL string
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

type OpenFinanceConfig struct {
	TransactionSyncStartDate string
}

type FirebaseConfig struct {
	CredentialsFile string
}

type TelemetryConfig struct {
	Enabled      bool
	ServiceName  string
	OTLPEndpoint string
}

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

	// Parse allowed hosts (comma-separated list)
	allowedHostsStr := getEnv("ALLOWED_HOSTS", "")
	var allowedHosts []string
	if allowedHostsStr != "" {
		for _, host := range strings.Split(allowedHostsStr, ",") {
			host = strings.TrimSpace(host)
			if host != "" {
				allowedHosts = append(allowedHosts, host)
			}
		}
	}

	// Construct OAuth callback URLs from HOST_URL
	hostURL := getEnv("HOST_URL", "")
	buildCallbackURL := func(path string, overrideEnv string) string {
		if override := getEnv(overrideEnv, ""); override != "" {
			return override
		}
		if hostURL != "" {
			return fmt.Sprintf("%s%s", hostURL, path)
		}
		return ""
	}

	googleWebURL := buildCallbackURL("/api/auth/oauth/callback", "GOOGLE_WEB_CALLBACK_URL")
	googleMobileURL := buildCallbackURL("/api/auth/oauth/mobile/callback", "GOOGLE_MOBILE_CALLBACK_URL")
	appleMobileURL := buildCallbackURL("/api/auth/oauth/apple/mobile/callback", "APPLE_MOBILE_CALLBACK_URL")

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("HOST", "0.0.0.0"),
			AllowedHosts: allowedHosts,
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
				ClientID:          getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret:      getEnv("GOOGLE_CLIENT_SECRET", ""),
				WebCallbackURL:    googleWebURL,
				MobileCallbackURL: googleMobileURL,
			},
			Apple: AppleOAuthConfig{
				TeamID:            getEnv("APPLE_TEAM_ID", ""),
				KeyID:             getEnv("APPLE_KEY_ID", ""),
				ClientID:          getEnv("APPLE_CLIENT_ID", ""),
				PrivateKeyPath:    getEnv("APPLE_PRIVATE_KEY_PATH", ""),
				MobileCallbackURL: appleMobileURL,
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
		OpenFinance: OpenFinanceConfig{
			TransactionSyncStartDate: getEnv("OPENFINANCE_TRANSACTION_SYNC_START_DATE", "2023-01-01"),
		},
		Firebase: FirebaseConfig{
			CredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", ""),
		},
		Telemetry: TelemetryConfig{
			Enabled:      getBoolEnv("OTEL_ENABLED", false),
			ServiceName:  getEnv("OTEL_SERVICE_NAME", "parsa-api"),
			OTLPEndpoint: getEnv("OTEL_EXPORTER_ENDPOINT", "localhost:4318"),
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
