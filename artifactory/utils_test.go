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

func TestUtilsMakeRequest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    *map[string]string
		authMethod string
		username   string
		password   string
		token      string
		wantErr    bool
	}{
		{
			name:       "Basic auth GET request",
			method:     "GET",
			path:       "/api/test",
			authMethod: "userPass",
			username:   "test",
			password:   "pass",
			wantErr:    false,
		},
		{
			name:       "Token auth POST request with body",
			method:     "POST",
			path:       "/api/test",
			body:       []byte(`{"key":"value"}`),
			authMethod: "accessToken",
			token:      "test-token",
			wantErr:    false,
		},
		{
			name:       "Request with custom headers",
			method:     "GET",
			path:       "/api/test",
			headers:    &map[string]string{"Custom-Header": "value"},
			authMethod: "userPass",
			username:   "test",
			password:   "pass",
			wantErr:    false,
		},
		{
			name:       "Invalid auth method",
			method:     "GET",
			path:       "/api/test",
			authMethod: "invalid",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server to verify the request
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != tt.method {
					t.Errorf("Method = %v, want %v", r.Method, tt.method)
				}

				// Verify auth headers
				switch tt.authMethod {
				case "userPass":
					username, password, ok := r.BasicAuth()
					if !ok || username != tt.username || password != tt.password {
						t.Error("Basic auth headers not set correctly")
					}
				case "accessToken":
					if auth := r.Header.Get("Authorization"); auth != "Bearer "+tt.token {
						t.Errorf("Token auth header = %v, want %v", auth, "Bearer "+tt.token)
					}
				}

				// Verify custom headers
				if tt.headers != nil {
					for k, v := range *tt.headers {
						if got := r.Header.Get(k); got != v {
							t.Errorf("Header %s = %v, want %v", k, got, v)
						}
					}
				}

				// Verify body
				if tt.body != nil {
					body, _ := io.ReadAll(r.Body)
					if !bytes.Equal(body, tt.body) {
						t.Errorf("Body = %v, want %v", string(body), string(tt.body))
					}
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			// Create client
			client := NewClient(&config.Config{
				ArtiScrapeURI: ts.URL,
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod:  tt.authMethod,
					Username:   tt.username,
					Password:   tt.password,
					AccessToken: tt.token,
				},
				Logger: slog.Default(),
			})

			// Make request
			var headersPtr *map[string]string
			var headersPtrPtr **map[string]string
			if tt.headers != nil {
				headersPtr = tt.headers
				headersPtrPtr = &headersPtr
			}
			resp, err := client.makeRequest(tt.method, ts.URL+tt.path, tt.body, headersPtrPtr)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp.StatusCode != http.StatusOK {
				t.Errorf("makeRequest() status = %v, want %v", resp.StatusCode, http.StatusOK)
			}
		})
	}
}

func TestUtilsHandleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		nodeID     string
		wantErr    bool
	}{
		{
			name:       "Successful response with node ID",
			statusCode: http.StatusOK,
			body:       `{"key":"value"}`,
			nodeID:     "node1",
			wantErr:    false,
		},
		{
			name:       "Not found error",
			statusCode: http.StatusNotFound,
			body:       `{"errors":["Resource not found"]}`,
			wantErr:    true,
		},
		{
			name:       "Server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"errors":["Internal server error"]}`,
			wantErr:    true,
		},
		{
			name:       "Invalid error response",
			statusCode: http.StatusBadRequest,
			body:       `{invalid json}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&config.Config{
				ArtiScrapeURI: "http://test",
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				Credentials: &config.Credentials{
					AuthMethod: "userPass",
					Username:  "test",
					Password:  "pass",
				},
				Logger: slog.Default(),
			})

			// Create response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(tt.body)),
				Header:     make(http.Header),
			}
			if tt.nodeID != "" {
				resp.Header.Set("x-artifactory-node-id", tt.nodeID)
			}

			// Handle response
			apiResp, err := client.handleResponse(resp, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("handleResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if apiResp.NodeId != tt.nodeID {
					t.Errorf("NodeId = %v, want %v", apiResp.NodeId, tt.nodeID)
				}
				if !bytes.Equal(apiResp.Body, []byte(tt.body)) {
					t.Errorf("Body = %v, want %v", string(apiResp.Body), tt.body)
				}
			}
		})
	}
}

func TestUtilsFetchHTTP(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful GET request",
			path:       "test",
			response:   `{"key":"value"}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "Failed GET request",
			path:       "test",
			response:   `{"errors":["Not found"]}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify it's a GET request
				if r.Method != "GET" {
					t.Errorf("Method = %v, want GET", r.Method)
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
					Username:   "test",
					Password:   "test",
				},
				Logger: slog.Default(),
			})

			resp, err := client.FetchHTTP(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchHTTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !bytes.Equal(resp.Body, []byte(tt.response)) {
				t.Errorf("Body = %v, want %v", string(resp.Body), tt.response)
			}
		})
	}
} 