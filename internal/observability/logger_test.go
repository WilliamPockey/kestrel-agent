package observability

import (
	"testing"

	"kestrel/internal/config"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.LogConfig
		wantErr bool
	}{
		{name: "json info", cfg: config.LogConfig{Level: "info", Format: "json"}},
		{name: "text debug", cfg: config.LogConfig{Level: "debug", Format: "text"}},
		{name: "invalid level", cfg: config.LogConfig{Level: "verbose", Format: "text"}, wantErr: true},
		{name: "invalid format", cfg: config.LogConfig{Level: "info", Format: "xml"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLogger(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewLogger() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
