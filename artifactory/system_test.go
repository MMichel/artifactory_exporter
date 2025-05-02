package artifactory

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peimanja/artifactory_exporter/config"
)

func TestFetchHealth(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantErr    bool
		wantHealth bool
		statusCode int
	}{
		{
			name:       "Healthy response",
			response:   "OK",
			wantErr:    false,
			wantHealth: true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Unhealthy response",
			response:   "Not OK",
			wantErr:    false,
			wantHealth: false,
			statusCode: http.StatusOK,
		},
		{
			name:       "Error response",
			response:   `{"errors": ["Internal server error"]}`,
			wantErr:    true,
			wantHealth: false,
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer ts.Close()

			client := NewClient(&config.Config{
				ArtiScrapeURI: ts.URL,
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod: "userPass",
					Username:   "test",
					Password:   "test",
				},
				Logger: slog.Default(),
			})

			health, err := client.FetchHealth()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchHealth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if health.Healthy != tt.wantHealth {
				t.Errorf("FetchHealth() health = %v, want %v", health.Healthy, tt.wantHealth)
			}
		})
	}
}

func TestLicenseInfo(t *testing.T) {
	tests := []struct {
		name         string
		licenseType  string
		validThrough string
		wantOSS      bool
		wantSeconds  int64
		wantErr      bool
	}{
		{
			name:         "Enterprise license",
			licenseType:  "Enterprise",
			validThrough: "Jan 1, 2025",
			wantOSS:     false,
			wantErr:     false,
		},
		{
			name:         "OSS license",
			licenseType:  "OSS",
			validThrough: "Jan 1, 2025",
			wantOSS:     true,
			wantErr:     false,
		},
		{
			name:         "Community Edition license",
			licenseType:  "Community Edition for C/C++",
			validThrough: "Jan 1, 2025",
			wantOSS:     true,
			wantErr:     false,
		},
		{
			name:         "Invalid date format",
			licenseType:  "Enterprise",
			validThrough: "2025-01-01",
			wantOSS:     false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := LicenseInfo{
				Type:         tt.licenseType,
				ValidThrough: tt.validThrough,
			}

			if got := license.IsOSS(); got != tt.wantOSS {
				t.Errorf("LicenseInfo.IsOSS() = %v, want %v", got, tt.wantOSS)
			}

			seconds, err := license.ValidSeconds()
			if (err != nil) != tt.wantErr {
				t.Errorf("LicenseInfo.ValidSeconds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.wantOSS {
				// Only check seconds for non-OSS licenses without errors
				expectedTime, _ := time.Parse(USAFullDate, tt.validThrough)
				expectedSeconds := expectedTime.Unix() - time.Now().Unix()
				if seconds != expectedSeconds {
					t.Errorf("LicenseInfo.ValidSeconds() = %v, want approximately %v", seconds, expectedSeconds)
				}
			}
		})
	}
}

func TestFetchLicense(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantErr    bool
		statusCode int
	}{
		{
			name: "Valid license response",
			response: `{
				"type": "Enterprise",
				"validThrough": "Jan 1, 2025",
				"licensedTo": "Test Company"
			}`,
			wantErr:    false,
			statusCode: http.StatusOK,
		},
		{
			name:       "Invalid JSON response",
			response:   `{"type": "Enterprise"`,
			wantErr:    true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Error response",
			response:   `{"errors": ["Internal server error"]}`,
			wantErr:    true,
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer ts.Close()

			client := NewClient(&config.Config{
				ArtiScrapeURI: ts.URL,
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod: "userPass",
					Username:   "test",
					Password:   "test",
				},
				Logger: slog.Default(),
			})

			license, err := client.FetchLicense()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchLicense() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var expected LicenseInfo
				if err := json.Unmarshal([]byte(tt.response), &expected); err != nil {
					t.Fatalf("Failed to unmarshal expected response: %v", err)
				}

				if license.Type != expected.Type {
					t.Errorf("License type = %v, want %v", license.Type, expected.Type)
				}
				if license.ValidThrough != expected.ValidThrough {
					t.Errorf("License validThrough = %v, want %v", license.ValidThrough, expected.ValidThrough)
				}
				if license.LicensedTo != expected.LicensedTo {
					t.Errorf("License licensedTo = %v, want %v", license.LicensedTo, expected.LicensedTo)
				}
			}
		})
	}
} 