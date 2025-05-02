package artifactory

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peimanja/artifactory_exporter/config"
)

func TestFetchAccessFederationValidStatus(t *testing.T) {
	tests := []struct {
		name              string
		pingResponse     string
		pingStatusCode   int
		validateResponse string
		validateStatusCode int
		federationTarget string
		wantStatus       bool
		wantErr         bool
	}{
		{
			name:            "Successful federation validation",
			pingResponse:    `{"nodeId":"test-node-id"}`,
			pingStatusCode:  http.StatusOK,
			validateResponse: `{"status":"ok"}`,
			validateStatusCode: http.StatusOK,
			federationTarget: "http://target",
			wantStatus:     true,
			wantErr:       false,
		},
		{
			name:            "Failed ping request",
			pingResponse:    `{"errors":["Not found"]}`,
			pingStatusCode:  http.StatusNotFound,
			federationTarget: "http://target",
			wantStatus:     false,
			wantErr:       true,
		},
		{
			name:            "Failed validation request",
			pingResponse:    `{"nodeId":"test-node-id"}`,
			pingStatusCode:  http.StatusOK,
			validateResponse: `{"errors":["Validation failed"]}`,
			validateStatusCode: http.StatusBadRequest,
			federationTarget: "http://target",
			wantStatus:     false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server that handles both ping and validation endpoints
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/artifactory/api/system/ping":
					w.WriteHeader(tt.pingStatusCode)
					w.Write([]byte(tt.pingResponse))
				case "/access/api/v1/system/federation/validate_server":
					w.WriteHeader(tt.validateStatusCode)
					w.Write([]byte(tt.validateResponse))
				default:
					t.Logf("Unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`{"errors":["Not found"]}`))
				}
			}))
			defer ts.Close()

			client := NewClient(&config.Config{
				ArtiScrapeURI:         ts.URL + "/artifactory",  // Add /artifactory suffix
				ArtiSSLVerify:         true,
				ArtiTimeout:           5 * time.Second,
				AccessFederationTarget: tt.federationTarget,
				Credentials: &config.Credentials{
					AuthMethod: "userPass",
					Username:  "test",
					Password:  "pass",
				},
				Logger: slog.Default(),
			})

			result, err := client.FetchAccessFederationValidStatus()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchAccessFederationValidStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.Status != tt.wantStatus {
				t.Errorf("FetchAccessFederationValidStatus() status = %v, want %v", result.Status, tt.wantStatus)
			}
		})
	}
} 