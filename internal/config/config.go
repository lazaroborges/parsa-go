package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	OAuth     OAuthConfig
	JWT       JWTConfig
	Scheduler SchedulerConfig
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

type SchedulerConfig struct {
	Enabled       bool
	ScheduleTimes []string
	WorkerCount   int
	JobDelay      time.Duration
	QueueSize     int
	RunOnStartup  bool
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
		Scheduler: SchedulerConfig{
			Enabled:       schedulerEnabled,
			ScheduleTimes: schedulerTimes,
			WorkerCount:   schedulerWorkers,
			JobDelay:      schedulerJobDelay,
			QueueSize:     schedulerQueueSize,
			RunOnStartup:  schedulerRunOnStartup,
		},
	}

	// Validate required fields
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
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
