package config

import (
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Application Settings
	AppName    string `env:"APP_NAME" envDefault:"go-gin-observability"`
	AppVersion string `env:"APP_VERSION" envDefault:"dev"`
	AppEnv     string `env:"APP_ENV" envDefault:"localhost"`
	Port       string `env:"GOLANG_PORT" envDefault:"8888"`

	// Security Settings
	JWTSecret  string `env:"JWT_SECRET" envDefault:"Template"`
	BcryptCost int    `env:"BCRYPT_COST" envDefault:"12"`

	// Database Settings
	DBHost               string `env:"DB_HOST" envDefault:"localhost"`
	DBPort               string `env:"DB_PORT" envDefault:"5432"`
	DBUser               string `env:"DB_USER" envDefault:"postgres"`
	DBPass               string `env:"DB_PASS" envDefault:"password"`
	DBName               string `env:"DB_NAME" envDefault:"gogintemplate"`
	DBMaxOpenConns       int    `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	DBMaxIdleConns       int    `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBConnMaxLifetimeMin int    `env:"DB_CONN_MAX_LIFETIME_MIN" envDefault:"0"`
	DBConnMaxIdleTimeMin int    `env:"DB_CONN_MAX_IDLE_TIME_MIN" envDefault:"0"`

	// Cache Configuration
	CacheTTLMinutes             int `env:"CACHE_TTL_MINUTES" envDefault:"5"`
	CacheCleanupIntervalMinutes int `env:"CACHE_CLEANUP_INTERVAL_MINUTES" envDefault:"10"`

	// Observability Settings
	OTELExporterEndpoint string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"alloy:4318"`
	OTELSamplingStrategy string  `env:"OTEL_SAMPLING_STRATEGY" envDefault:"ratio"`
	OTELSamplingRate     float64 `env:"OTEL_SAMPLING_RATE" envDefault:"0.1"`

	// Logging Settings
	EnableStdoutLogs  bool   `env:"ENABLE_STDOUT_LOGS" envDefault:"true"`
	EnableOTLPLogs    bool   `env:"ENABLE_OTLP_LOGS" envDefault:"true"`
	LogBufferSize     int    `env:"LOG_BUFFER_SIZE" envDefault:"5000"`
	LogDropOnFull     bool   `env:"LOG_DROP_ON_FULL" envDefault:"true"`
	LogBlacklistPaths string `env:"LOG_BLACKLIST_PATHS" envDefault:""`

	// Profiling Settings
	EnableProfiling     bool   `env:"ENABLE_PROFILING" envDefault:"true"`
	PyroscopeServerAddr string `env:"PYROSCOPE_SERVER_ADDRESS" envDefault:"http://pyroscope:4040"`

	// Performance Configuration
	MetricsCollectionIntervalSeconds int `env:"METRICS_COLLECTION_INTERVAL_SECONDS" envDefault:"15"`
}

var appConfig *Config

// Load loads configuration from environment variables
// It always reloads the configuration (useful for tests)
func Load() *Config {
	// Load .env file if not running in docker
	if os.Getenv("APP_ENV") != "docker" {
		_ = godotenv.Load(".env")
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	// Validate sampling rate
	if cfg.OTELSamplingRate < 0 || cfg.OTELSamplingRate > 1 {
		cfg.OTELSamplingRate = 0.1
	}

	// Enforce minimum bcrypt cost for security
	if cfg.BcryptCost < 10 {
		cfg.BcryptCost = 10
	}
	if cfg.BcryptCost > 31 {
		cfg.BcryptCost = 31
	}

	appConfig = cfg
	return cfg
}

// Reset resets the configuration cache (useful for testing)
func Reset() {
	appConfig = nil
}

// Get returns the loaded configuration
func Get() *Config {
	if appConfig == nil {
		return Load()
	}
	return appConfig
}

// Convenience getters for commonly used values

func (c *Config) CacheTTL() time.Duration {
	return time.Duration(c.CacheTTLMinutes) * time.Minute
}

func (c *Config) CacheCleanupInterval() time.Duration {
	return time.Duration(c.CacheCleanupIntervalMinutes) * time.Minute
}

func (c *Config) MetricsCollectionInterval() time.Duration {
	return time.Duration(c.MetricsCollectionIntervalSeconds) * time.Second
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "dev" || c.AppEnv == "development"
}

func (c *Config) IsLocalhost() bool {
	return c.AppEnv == "localhost"
}

func (c *Config) DBConnMaxLifetime() time.Duration {
	if c.DBConnMaxLifetimeMin > 0 {
		return time.Duration(c.DBConnMaxLifetimeMin) * time.Minute
	}
	return 0
}

func (c *Config) DBConnMaxIdleTime() time.Duration {
	if c.DBConnMaxIdleTimeMin > 0 {
		return time.Duration(c.DBConnMaxIdleTimeMin) * time.Minute
	}
	return 0
}
