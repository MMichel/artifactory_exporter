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

func TestFetchStorageInfo(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		wantErr     bool
		statusCode  int
		nodeId      string
	}{
		{
			name: "Successful response",
			response: `{
				"binariesSummary": {
					"binariesCount": "100",
					"binariesSize": "1024",
					"artifactsSize": "2048",
					"optimization": "50%",
					"itemsCount": "200",
					"artifactsCount": "150"
				},
				"fileStoreSummary": {
					"storageType": "file-system",
					"storageDirectory": "/data",
					"totalSpace": "1000000",
					"usedSpace": "500000",
					"freeSpace": "500000"
				},
				"repositoriesSummaryList": [
					{
						"repoKey": "test-repo",
						"repoType": "local",
						"foldersCount": 10,
						"filesCount": 100,
						"usedSpace": "100000",
						"itemsCount": 110,
						"packageType": "generic",
						"percentage": "10%"
					}
				]
			}`,
			wantErr:    false,
			statusCode: http.StatusOK,
			nodeId:     "node1",
		},
		{
			name: "Invalid JSON response",
			response: `{
				"binariesSummary": {
					"binariesCount": "100",
					"binariesSize": "1024"
				}
				"fileStoreSummary": {
					"storageType": "file-system"
				}
			}`,
			wantErr:    true,
			statusCode: http.StatusOK,
			nodeId:     "node1",
		},
		{
			name:       "API error response",
			response:   `{"errors": ["Internal server error"]}`,
			wantErr:    true,
			statusCode: http.StatusInternalServerError,
			nodeId:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("x-artifactory-node-id", tt.nodeId)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer ts.Close()

			// Create a client with the test server URL
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

			// Test FetchStorageInfo
			info, err := client.FetchStorageInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchStorageInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the NodeId
				if info.NodeId != tt.nodeId {
					t.Errorf("NodeId = %v, want %v", info.NodeId, tt.nodeId)
				}

				// Verify the response was parsed correctly
				var expected StorageInfo
				if err := json.Unmarshal([]byte(tt.response), &expected); err != nil {
					t.Fatalf("Failed to unmarshal expected response: %v", err)
				}

				// Compare the parsed values
				if info.BinariesSummary.BinariesCount != expected.BinariesSummary.BinariesCount {
					t.Errorf("BinariesCount = %v, want %v", 
						info.BinariesSummary.BinariesCount, 
						expected.BinariesSummary.BinariesCount)
				}

				if info.FileStoreSummary.StorageType != expected.FileStoreSummary.StorageType {
					t.Errorf("StorageType = %v, want %v", 
						info.FileStoreSummary.StorageType, 
						expected.FileStoreSummary.StorageType)
				}

				if len(info.RepositoriesSummaryList) != len(expected.RepositoriesSummaryList) {
					t.Errorf("RepositoriesSummaryList length = %v, want %v", 
						len(info.RepositoriesSummaryList), 
						len(expected.RepositoriesSummaryList))
				}
			}
		})
	}
} 