package artifactory

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peimanja/artifactory_exporter/config"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "Valid config with basic auth",
			config: &config.Config{
				ArtiScrapeURI: "http://localhost:8081",
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod: "userPass",
					Username:   "test",
					Password:   "test",
				},
				Logger: slog.Default(),
			},
			wantErr: false,
		},
		{
			name: "Valid config with access token",
			config: &config.Config{
				ArtiScrapeURI: "http://localhost:8081",
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod:  "accessToken",
					AccessToken: "test-token",
				},
				Logger: slog.Default(),
			},
			wantErr: false,
		},
		{
			name: "Invalid auth method",
			config: &config.Config{
				ArtiScrapeURI: "http://localhost:8081",
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod: "invalid",
				},
				Logger: slog.Default(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			
			// Test makeRequest with a mock server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantErr {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				
				// Verify auth headers
				if tt.config.Credentials.AuthMethod == "userPass" {
					username, password, ok := r.BasicAuth()
					if !ok || username != tt.config.Credentials.Username || password != tt.config.Credentials.Password {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
				} else if tt.config.Credentials.AuthMethod == "accessToken" {
					auth := r.Header.Get("Authorization")
					if auth != "Bearer "+tt.config.Credentials.AccessToken {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
				}
				
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{}"))
			}))
			defer ts.Close()
			
			// Override URI for testing
			client.URI = ts.URL
			
			// Test FetchHTTP
			_, err := client.FetchHTTP("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchHTTP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "Successful response",
			statusCode: http.StatusOK,
			body:       `{"key": "value"}`,
			wantErr:    false,
		},
		{
			name:       "Not found response",
			statusCode: http.StatusNotFound,
			body:       `{"errors": ["Not found"]}`,
			wantErr:    true,
		},
		{
			name:       "Server error response",
			statusCode: http.StatusInternalServerError,
			body:       `{"errors": ["Internal server error"]}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&config.Config{
				ArtiScrapeURI: "http://localhost:8081",
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod: "userPass",
					Username:   "test",
					Password:   "test",
				},
				Logger: slog.Default(),
			})
			
			// Create a mock response with proper body
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(tt.body)),
			}
			
			// Test handleResponse
			_, err := client.handleResponse(resp, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("handleResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
} 