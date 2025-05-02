package artifactory

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peimanja/artifactory_exporter/config"
)

func TestFetchOpenMetrics(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		statusCode   int
		nodeId       string
		wantMetrics  string
		wantErr      bool
	}{
		{
			name: "Successful response",
			response: `# HELP jfrog_artifactory_artifact_size_bytes Size of artifact.
# TYPE jfrog_artifactory_artifact_size_bytes gauge
jfrog_artifactory_artifact_size_bytes{repo="libs-release-local"} 1024
# HELP jfrog_artifactory_request_duration_seconds Request duration in seconds.
# TYPE jfrog_artifactory_request_duration_seconds histogram
jfrog_artifactory_request_duration_seconds_bucket{le="0.1"} 100`,
			statusCode:   http.StatusOK,
			nodeId:       "node1",
			wantMetrics:  `# HELP jfrog_artifactory_artifact_size_bytes Size of artifact.
# TYPE jfrog_artifactory_artifact_size_bytes gauge
jfrog_artifactory_artifact_size_bytes{repo="libs-release-local"} 1024
# HELP jfrog_artifactory_request_duration_seconds Request duration in seconds.
# TYPE jfrog_artifactory_request_duration_seconds histogram
jfrog_artifactory_request_duration_seconds_bucket{le="0.1"} 100`,
			wantErr:      false,
		},
		{
			name:         "Empty response",
			response:     ``,
			statusCode:   http.StatusOK,
			nodeId:       "node1",
			wantMetrics:  ``,
			wantErr:      false,
		},
		{
			name:         "Not found response",
			response:     `{"errors":["Not found"]}`,
			statusCode:   http.StatusNotFound,
			nodeId:       "",
			wantMetrics:  "",
			wantErr:      false,
		},
		{
			name:         "Server error",
			response:     `{"errors":["Internal server error"]}`,
			statusCode:   http.StatusInternalServerError,
			nodeId:       "",
			wantMetrics:  "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/metrics" {
					t.Errorf("Expected path %s, got %s", "/api/v1/metrics", r.URL.Path)
				}
				w.Header().Set("x-artifactory-node-id", tt.nodeId)
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
					Username:  "test",
					Password:  "pass",
				},
				Logger: slog.Default(),
			})

			got, err := client.FetchOpenMetrics()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchOpenMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.NodeId != tt.nodeId {
					t.Errorf("FetchOpenMetrics() nodeId = %v, want %v", got.NodeId, tt.nodeId)
				}
				if got.PromMetrics != tt.wantMetrics {
					t.Errorf("FetchOpenMetrics() metrics = %v, want %v", got.PromMetrics, tt.wantMetrics)
				}
			}
		})
	}
} 