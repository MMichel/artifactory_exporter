package artifactory

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peimanja/artifactory_exporter/config"
)

func TestFetchUsers(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		nodeId     string
		wantUsers  []User
		wantErr    bool
	}{
		{
			name: "Successful response with users",
			response: `[
				{"name": "user1", "realm": "internal"},
				{"name": "user2", "realm": "ldap"}
			]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantUsers: []User{
				{Name: "user1", Realm: "internal"},
				{Name: "user2", Realm: "ldap"},
			},
			wantErr: false,
		},
		{
			name:       "Empty response",
			response:   `[]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantUsers:  []User{},
			wantErr:    false,
		},
		{
			name:       "Invalid JSON response",
			response:   `invalid json`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantUsers:  nil,
			wantErr:    true,
		},
		{
			name:       "API error response",
			response:   `{"errors": ["Internal server error"]}`,
			statusCode: http.StatusInternalServerError,
			nodeId:     "",
			wantUsers:  nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/security/users" {
					t.Errorf("Expected path %s, got %s", "/api/security/users", r.URL.Path)
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

			got, err := client.FetchUsers()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.NodeId != tt.nodeId {
					t.Errorf("FetchUsers() nodeId = %v, want %v", got.NodeId, tt.nodeId)
				}
				if len(got.Users) != len(tt.wantUsers) {
					t.Errorf("FetchUsers() got %v users, want %v", len(got.Users), len(tt.wantUsers))
					return
				}
				for i, user := range got.Users {
					if user != tt.wantUsers[i] {
						t.Errorf("FetchUsers() user[%d] = %v, want %v", i, user, tt.wantUsers[i])
					}
				}
			}
		})
	}
}

func TestFetchGroups(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		statusCode  int
		nodeId      string
		wantGroups  []Group
		wantErr     bool
	}{
		{
			name: "Successful response with groups",
			response: `[
				{"name": "group1", "uri": "internal"},
				{"name": "group2", "uri": "ldap"}
			]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantGroups: []Group{
				{Name: "group1", Realm: "internal"},
				{Name: "group2", Realm: "ldap"},
			},
			wantErr: false,
		},
		{
			name:       "Empty response",
			response:   `[]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantGroups: []Group{},
			wantErr:    false,
		},
		{
			name:       "Invalid JSON response",
			response:   `invalid json`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantGroups: nil,
			wantErr:    true,
		},
		{
			name:       "API error response",
			response:   `{"errors": ["Internal server error"]}`,
			statusCode: http.StatusInternalServerError,
			nodeId:     "",
			wantGroups: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/security/groups" {
					t.Errorf("Expected path %s, got %s", "/api/security/groups", r.URL.Path)
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

			got, err := client.FetchGroups()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.NodeId != tt.nodeId {
					t.Errorf("FetchGroups() nodeId = %v, want %v", got.NodeId, tt.nodeId)
				}
				if len(got.Groups) != len(tt.wantGroups) {
					t.Errorf("FetchGroups() got %v groups, want %v", len(got.Groups), len(tt.wantGroups))
					return
				}
				for i, group := range got.Groups {
					if group != tt.wantGroups[i] {
						t.Errorf("FetchGroups() group[%d] = %v, want %v", i, group, tt.wantGroups[i])
					}
				}
			}
		})
	}
}

func TestFetchCertificates(t *testing.T) {
	tests := []struct {
		name            string
		response        string
		statusCode      int
		nodeId          string
		wantCertificates []Certificate
		wantErr         bool
	}{
		{
			name: "Successful response with certificates",
			response: `[
				{
					"certificateAlias": "cert1",
					"issuedTo": "example.com",
					"issuedBy": "CA",
					"issuedOn": "2024-01-01",
					"validUntil": "2025-01-01",
					"fingerprint": "abc123"
				},
				{
					"certificateAlias": "cert2",
					"issuedTo": "test.com",
					"issuedBy": "CA",
					"issuedOn": "2024-02-01",
					"validUntil": "2025-02-01",
					"fingerprint": "def456"
				}
			]`,
			statusCode: http.StatusOK,
			nodeId:     "node1",
			wantCertificates: []Certificate{
				{
					CertificateAlias: "cert1",
					IssuedTo:        "example.com",
					IssuedBy:        "CA",
					IssuedOn:        "2024-01-01",
					ValidUntil:      "2025-01-01",
					Fingerprint:     "abc123",
				},
				{
					CertificateAlias: "cert2",
					IssuedTo:        "test.com",
					IssuedBy:        "CA",
					IssuedOn:        "2024-02-01",
					ValidUntil:      "2025-02-01",
					Fingerprint:     "def456",
				},
			},
			wantErr: false,
		},
		{
			name:            "Empty response",
			response:        `[]`,
			statusCode:      http.StatusOK,
			nodeId:          "node1",
			wantCertificates: []Certificate{},
			wantErr:         false,
		},
		{
			name:            "Invalid JSON response",
			response:        `invalid json`,
			statusCode:      http.StatusOK,
			nodeId:          "node1",
			wantCertificates: nil,
			wantErr:         true,
		},
		{
			name:            "API error response",
			response:        `{"errors": ["Internal server error"]}`,
			statusCode:      http.StatusInternalServerError,
			nodeId:          "",
			wantCertificates: nil,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/system/security/certificates" {
					t.Errorf("Expected path %s, got %s", "/api/system/security/certificates", r.URL.Path)
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

			got, err := client.FetchCertificates()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchCertificates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.NodeId != tt.nodeId {
					t.Errorf("FetchCertificates() nodeId = %v, want %v", got.NodeId, tt.nodeId)
				}
				if len(got.Certificates) != len(tt.wantCertificates) {
					t.Errorf("FetchCertificates() got %v certificates, want %v", len(got.Certificates), len(tt.wantCertificates))
					return
				}
				for i, cert := range got.Certificates {
					if cert != tt.wantCertificates[i] {
						t.Errorf("FetchCertificates() certificate[%d] = %v, want %v", i, cert, tt.wantCertificates[i])
					}
				}
			}
		})
	}
} 