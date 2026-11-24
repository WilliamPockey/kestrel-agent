package config

import (
	"strings"
	"testing"
	"time"
)

var envKeys = []string{
	"KESTREL_ENV",
	"KESTREL_HTTP_ADDR",
	"KESTREL_HTTP_READ_HEADER_TIMEOUT",
	"KESTREL_HTTP_READ_TIMEOUT",
	"KESTREL_HTTP_WRITE_TIMEOUT",
	"KESTREL_HTTP_IDLE_TIMEOUT",
	"KESTREL_HTTP_SHUTDOWN_TIMEOUT",
	"KESTREL_LOG_LEVEL",
	"KESTREL_LOG_FORMAT",
}

func setEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range envKeys {
		t.Setenv(key, "")
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want dev", cfg.Env)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.HTTP.WriteTimeout != 0 {
		t.Errorf("HTTP.WriteTimeout = %v, want 0", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.ShutdownTimeout != 10*time.Second {
		t.Errorf("HTTP.ShutdownTimeout = %v, want 10s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want text", cfg.Log.Format)
	}
}

func TestLoadOverrides(t *testing.T) {
	setEnv(t, map[string]string{
		"KESTREL_ENV":                      "test",
		"KESTREL_HTTP_ADDR":                "127.0.0.1:9090",
		"KESTREL_HTTP_READ_HEADER_TIMEOUT": "2s",
		"KESTREL_HTTP_READ_TIMEOUT":        "3s",
		"KESTREL_HTTP_WRITE_TIMEOUT":       "4s",
		"KESTREL_HTTP_IDLE_TIMEOUT":        "5s",
		"KESTREL_HTTP_SHUTDOWN_TIMEOUT":    "6s",
		"KESTREL_LOG_LEVEL":                "debug",
		"KESTREL_LOG_FORMAT":               "json",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Addr != "127.0.0.1:9090" {
		t.Errorf("HTTP.Addr = %q", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadHeaderTimeout != 2*time.Second {
		t.Errorf("HTTP.ReadHeaderTimeout = %v", cfg.HTTP.ReadHeaderTimeout)
	}
	if cfg.HTTP.ReadTimeout != 3*time.Second {
		t.Errorf("HTTP.ReadTimeout = %v", cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 4*time.Second {
		t.Errorf("HTTP.WriteTimeout = %v", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.IdleTimeout != 5*time.Second {
		t.Errorf("HTTP.IdleTimeout = %v", cfg.HTTP.IdleTimeout)
	}
	if cfg.HTTP.ShutdownTimeout != 6*time.Second {
		t.Errorf("HTTP.ShutdownTimeout = %v", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q", cfg.Log.Format)
	}
}

func TestLoadProdDefaultsToJSONLogs(t *testing.T) {
	setEnv(t, map[string]string{
		"KESTREL_ENV": "prod",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", cfg.Log.Format)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	setEnv(t, map[string]string{
		"KESTREL_HTTP_READ_TIMEOUT": "soon",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted an invalid duration")
	}
	if !strings.Contains(err.Error(), "KESTREL_HTTP_READ_TIMEOUT") {
		t.Fatalf("error = %v, want env key in message", err)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	setEnv(t, map[string]string{
		"KESTREL_LOG_LEVEL": "verbose",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted an invalid log level")
	}
}

func TestLoadRejectsNegativeWriteTimeout(t *testing.T) {
	setEnv(t, map[string]string{
		"KESTREL_HTTP_WRITE_TIMEOUT": "-1s",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted a negative write timeout")
	}
}

func TestValidateRejectsBlankAddr(t *testing.T) {
	cfg := Default()
	cfg.HTTP.Addr = "   "
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted a blank address")
	}
}

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("Default().Validate() = %v", err)
	}
}
