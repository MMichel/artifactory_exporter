package logger

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantType string
	}{
		{
			name: "Default config creates text logger",
			config: Config{
				Format: "",
				Level:  "",
			},
			wantType: "text",
		},
		{
			name: "JSON format creates JSON logger",
			config: Config{
				Format: fmtJSON,
				Level:  lvlFNameInfo,
			},
			wantType: "json",
		},
		{
			name: "Text format creates text logger",
			config: Config{
				Format: fmtTXT,
				Level:  lvlFNameInfo,
			},
			wantType: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			var buf bytes.Buffer
			
			// Create a new logger with the buffer as output
			var handler slog.Handler
			if tt.wantType == "json" {
				handler = slog.NewJSONHandler(&buf, &slog.HandlerOptions{
					Level: lvlFromConfig(tt.config),
				})
			} else {
				handler = slog.NewTextHandler(&buf, &slog.HandlerOptions{
					Level: lvlFromConfig(tt.config),
				})
			}
			logger := slog.New(handler)
			
			// Log a test message
			logger.Info("test message")
			
			// Check the output format
			output := buf.Bytes()
			if tt.wantType == "json" && !bytes.Contains(output, []byte(`"msg":"test message"`)) {
				t.Errorf("Expected JSON format output, got: %s", string(output))
			} else if tt.wantType == "text" && !bytes.Contains(output, []byte("test message")) {
				t.Errorf("Expected text format output, got: %s", string(output))
			}
		})
	}
}

func TestLevelFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantLevel slog.Level
	}{
		{
			name:     "Default level",
			config:   Config{Level: ""},
			wantLevel: slog.LevelInfo,
		},
		{
			name:     "Debug level",
			config:   Config{Level: lvlFNameDebug},
			wantLevel: slog.LevelDebug,
		},
		{
			name:     "Info level",
			config:   Config{Level: lvlFNameInfo},
			wantLevel: slog.LevelInfo,
		},
		{
			name:     "Warn level",
			config:   Config{Level: lvlFNameWarn},
			wantLevel: slog.LevelWarn,
		},
		{
			name:     "Error level",
			config:   Config{Level: lvlFNameError},
			wantLevel: slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lvlFromConfig(tt.config)
			if got != tt.wantLevel {
				t.Errorf("lvlFromConfig() = %v, want %v", got, tt.wantLevel)
			}
		})
	}
}

func TestFormatFromConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name:   "Default format",
			config: Config{Format: ""},
			want:   FormatDefault,
		},
		{
			name:   "JSON format",
			config: Config{Format: fmtJSON},
			want:   fmtJSON,
		},
		{
			name:   "Text format",
			config: Config{Format: fmtTXT},
			want:   fmtTXT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fmtFromConfig(tt.config); got != tt.want {
				t.Errorf("fmtFromConfig() = %v, want %v", got, tt.want)
			}
		})
	}
} 