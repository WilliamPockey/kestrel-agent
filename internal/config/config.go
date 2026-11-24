package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env  string
	HTTP HTTPConfig
	Log  LogConfig
}

type HTTPConfig struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

func Default() Config {
	return Config{
		Env: "dev",
		HTTP: HTTPConfig{
			Addr:              ":8080",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      0,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   10 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func Load() (Config, error) {
	cfg := Default()
	cfg.Env = env("KESTREL_ENV", cfg.Env)
	cfg.HTTP.Addr = env("KESTREL_HTTP_ADDR", cfg.HTTP.Addr)

	var err error
	if cfg.HTTP.ReadHeaderTimeout, err = envDuration("KESTREL_HTTP_READ_HEADER_TIMEOUT", cfg.HTTP.ReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ReadTimeout, err = envDuration("KESTREL_HTTP_READ_TIMEOUT", cfg.HTTP.ReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.WriteTimeout, err = envDuration("KESTREL_HTTP_WRITE_TIMEOUT", cfg.HTTP.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.IdleTimeout, err = envDuration("KESTREL_HTTP_IDLE_TIMEOUT", cfg.HTTP.IdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ShutdownTimeout, err = envDuration("KESTREL_HTTP_SHUTDOWN_TIMEOUT", cfg.HTTP.ShutdownTimeout); err != nil {
		return Config{}, err
	}

	cfg.Log.Level = env("KESTREL_LOG_LEVEL", cfg.Log.Level)
	cfg.Log.Format = env("KESTREL_LOG_FORMAT", defaultLogFormat(cfg.Env))

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Env) == "" {
		return errors.New("env must not be empty")
	}
	if strings.TrimSpace(c.HTTP.Addr) == "" {
		return errors.New("http.addr must not be empty")
	}
	if c.HTTP.ReadHeaderTimeout <= 0 {
		return errors.New("http.read_header_timeout must be positive")
	}
	if c.HTTP.ReadTimeout <= 0 {
		return errors.New("http.read_timeout must be positive")
	}
	if c.HTTP.WriteTimeout < 0 {
		return errors.New("http.write_timeout must not be negative")
	}
	if c.HTTP.IdleTimeout <= 0 {
		return errors.New("http.idle_timeout must be positive")
	}
	if c.HTTP.ShutdownTimeout <= 0 {
		return errors.New("http.shutdown_timeout must be positive")
	}
	switch c.Log.Format {
	case "json", "text":
	default:
		return fmt.Errorf("unsupported log format %q", c.Log.Format)
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unsupported log level %q", c.Log.Level)
	}
	return nil
}

func defaultLogFormat(env string) string {
	if env == "prod" || env == "production" {
		return "json"
	}
	return "text"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return value, nil
}
