package artifactory

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peimanja/artifactory_exporter/config"
)

func TestIsFederationEnabled(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		want       bool
	}{
		{
			name:       "Federation enabled",
			response:   `[]`,
			statusCode: http.StatusOK,
			want:       true,
		},
		{
			name:       "Federation disabled",
			response:   `{"errors":["Not found"]}`,
			statusCode: http.StatusNotFound,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/federation/status/unavailableMirrors" {
					t.Errorf("Expected path %s, got %s", "/api/federation/status/unavailableMirrors", r.URL.Path)
				}
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

			if got := client.IsFederationEnabled(); got != tt.want {
				t.Errorf("IsFederationEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchMirrorLags(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		nodeId     string
		wantLags   []MirrorLag
		wantErr    bool
	}{
		{
			name: "Successful response with lags",
			response: `[
				{
					"localRepoKey": "repo1",
					"remoteUrl": "http://remote1",
					"remoteRepoKey": "remote1",
					"lagInMS": 1000
				},
				{
					"localRepoKey": "repo2",
					"remoteUrl": "http://remote2",
					"remoteRepoKey": "remote2",
					"lagInMS": 2000
				}
			]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantLags: []MirrorLag{
				{
					LocalRepoKey:  "repo1",
					RemoteUrl:     "http://remote1",
					RemoteRepoKey: "remote1",
					LagInMS:       1000,
				},
				{
					LocalRepoKey:  "repo2",
					RemoteUrl:     "http://remote2",
					RemoteRepoKey: "remote2",
					LagInMS:       2000,
				},
			},
			wantErr: false,
		},
		{
			name:       "No lags found",
			response:   `[]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantLags:   []MirrorLag{},
			wantErr:    false,
		},
		{
			name:       "Invalid JSON response",
			response:   `invalid json`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantLags:   nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/federation/status/mirrorsLag" {
					t.Errorf("Expected path %s, got %s", "/api/federation/status/mirrorsLag", r.URL.Path)
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

			got, err := client.FetchMirrorLags()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchMirrorLags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.NodeId != tt.nodeId {
					t.Errorf("FetchMirrorLags() nodeId = %v, want %v", got.NodeId, tt.nodeId)
				}
				if len(got.MirrorLags) != len(tt.wantLags) {
					t.Errorf("FetchMirrorLags() got %v lags, want %v", len(got.MirrorLags), len(tt.wantLags))
					return
				}
				for i, lag := range got.MirrorLags {
					if lag != tt.wantLags[i] {
						t.Errorf("FetchMirrorLags() lag[%d] = %v, want %v", i, lag, tt.wantLags[i])
					}
				}
			}
		})
	}
}

func TestFetchUnavailableMirrors(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		statusCode   int
		nodeId       string
		wantMirrors  []UnavailableMirror
		wantErr      bool
	}{
		{
			name: "Successful response with unavailable mirrors",
			response: `[
				{
					"localRepoKey": "repo1",
					"remoteUrl": "http://remote1",
					"remoteRepoKey": "remote1",
					"status": "offline"
				},
				{
					"localRepoKey": "repo2",
					"remoteUrl": "http://remote2",
					"remoteRepoKey": "remote2",
					"status": "error"
				}
			]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantMirrors: []UnavailableMirror{
				{
					LocalRepoKey:  "repo1",
					RemoteUrl:     "http://remote1",
					RemoteRepoKey: "remote1",
					Status:        "offline",
				},
				{
					LocalRepoKey:  "repo2",
					RemoteUrl:     "http://remote2",
					RemoteRepoKey: "remote2",
					Status:        "error",
				},
			},
			wantErr: false,
		},
		{
			name:         "No unavailable mirrors",
			response:     `[]`,
			statusCode:   http.StatusOK,
			nodeId:       "node1",
			wantMirrors:  []UnavailableMirror{},
			wantErr:      false,
		},
		{
			name:         "Invalid JSON response",
			response:     `invalid json`,
			statusCode:   http.StatusOK,
			nodeId:       "node1",
			wantMirrors:  nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/federation/status/unavailableMirrors" {
					t.Errorf("Expected path %s, got %s", "/api/federation/status/unavailableMirrors", r.URL.Path)
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

			got, err := client.FetchUnavailableMirrors()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchUnavailableMirrors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.NodeId != tt.nodeId {
					t.Errorf("FetchUnavailableMirrors() nodeId = %v, want %v", got.NodeId, tt.nodeId)
				}
				if len(got.UnavailableMirrors) != len(tt.wantMirrors) {
					t.Errorf("FetchUnavailableMirrors() got %v mirrors, want %v", len(got.UnavailableMirrors), len(tt.wantMirrors))
					return
				}
				for i, mirror := range got.UnavailableMirrors {
					if mirror != tt.wantMirrors[i] {
						t.Errorf("FetchUnavailableMirrors() mirror[%d] = %v, want %v", i, mirror, tt.wantMirrors[i])
					}
				}
			}
		})
	}
} 