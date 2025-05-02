package artifactory

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peimanja/artifactory_exporter/config"
)

func TestFetchReplications(t *testing.T) {
	tests := []struct {
		name                string
		replicationsResp   string
		replicationsStatus int
		statusResp         string
		statusStatus       int
		nodeId             string
		wantReplications   []Replication
		wantErr            bool
		replicationStatus  bool
	}{
		{
			name: "Successful response with replication status disabled",
			replicationsResp: `[
				{
					"replicationType": "push",
					"enabled": true,
					"cronExp": "0 0 * * * ?",
					"syncDeletes": true,
					"syncProperties": true,
					"pathPrefix": "",
					"repoKey": "repo1",
					"url": "http://target1",
					"enableEventReplication": false,
					"checkBinaryExistenceInFilestore": false,
					"syncStatistics": false,
					"status": ""
				},
				{
					"replicationType": "pull",
					"enabled": false,
					"cronExp": "0 0 * * * ?",
					"syncDeletes": false,
					"syncProperties": true,
					"pathPrefix": "path/",
					"repoKey": "repo2",
					"url": "http://target2",
					"enableEventReplication": true,
					"checkBinaryExistenceInFilestore": true,
					"syncStatistics": true,
					"status": ""
				}
			]`,
			replicationsStatus: http.StatusOK,
			nodeId:             "node1",
			wantReplications: []Replication{
				{
					ReplicationType:                 "push",
					Enabled:                         true,
					CronExp:                        "0 0 * * * ?",
					SyncDeletes:                    true,
					SyncProperties:                 true,
					PathPrefix:                     "",
					RepoKey:                        "repo1",
					URL:                            "http://target1",
					EnableEventReplication:         false,
					CheckBinaryExistenceInFilestore: false,
					SyncStatistics:                 false,
					Status:                         "",
				},
				{
					ReplicationType:                 "pull",
					Enabled:                         false,
					CronExp:                        "0 0 * * * ?",
					SyncDeletes:                    false,
					SyncProperties:                 true,
					PathPrefix:                     "path/",
					RepoKey:                        "repo2",
					URL:                            "http://target2",
					EnableEventReplication:         true,
					CheckBinaryExistenceInFilestore: true,
					SyncStatistics:                 true,
					Status:                         "",
				},
			},
			wantErr:           false,
			replicationStatus: false,
		},
		{
			name: "Successful response with replication status enabled",
			replicationsResp: `[
				{
					"replicationType": "push",
					"enabled": true,
					"cronExp": "0 0 * * * ?",
					"syncDeletes": true,
					"syncProperties": true,
					"pathPrefix": "",
					"repoKey": "repo1",
					"url": "http://target1",
					"enableEventReplication": false,
					"checkBinaryExistenceInFilestore": false,
					"syncStatistics": false,
					"status": ""
				}
			]`,
			replicationsStatus: http.StatusOK,
			statusResp:         `{"status":"ok"}`,
			statusStatus:       http.StatusOK,
			nodeId:             "node1",
			wantReplications: []Replication{
				{
					ReplicationType:                 "push",
					Enabled:                         true,
					CronExp:                        "0 0 * * * ?",
					SyncDeletes:                    true,
					SyncProperties:                 true,
					PathPrefix:                     "",
					RepoKey:                        "repo1",
					URL:                            "http://target1",
					EnableEventReplication:         false,
					CheckBinaryExistenceInFilestore: false,
					SyncStatistics:                 false,
					Status:                         "ok",
				},
			},
			wantErr:           false,
			replicationStatus: true,
		},
		{
			name:                "Not found response",
			replicationsResp:   `{"errors":["Not found"]}`,
			replicationsStatus: http.StatusNotFound,
			nodeId:             "",
			wantReplications:   nil,
			wantErr:            false,
			replicationStatus:  false,
		},
		{
			name:                "Invalid JSON response",
			replicationsResp:   `invalid json`,
			replicationsStatus: http.StatusOK,
			nodeId:             "node1",
			wantReplications:   nil,
			wantErr:            true,
			replicationStatus:  false,
		},
		{
			name: "Invalid status JSON response",
			replicationsResp: `[
				{
					"replicationType": "push",
					"enabled": true,
					"cronExp": "0 0 * * * ?",
					"syncDeletes": true,
					"syncProperties": true,
					"pathPrefix": "",
					"repoKey": "repo1",
					"url": "http://target1",
					"enableEventReplication": false,
					"checkBinaryExistenceInFilestore": false,
					"syncStatistics": false,
					"status": ""
				}
			]`,
			replicationsStatus: http.StatusOK,
			statusResp:         `invalid json`,
			statusStatus:       http.StatusOK,
			nodeId:             "node1",
			wantReplications:   nil,
			wantErr:            true,
			replicationStatus:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/replications":
					w.Header().Set("x-artifactory-node-id", tt.nodeId)
					w.WriteHeader(tt.replicationsStatus)
					w.Write([]byte(tt.replicationsResp))
				case "/api/replication/repo1":
					w.WriteHeader(tt.statusStatus)
					w.Write([]byte(tt.statusResp))
				default:
					t.Errorf("Unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`{"errors":["Not found"]}`))
				}
			}))
			defer ts.Close()

			client := NewClient(&config.Config{
				ArtiScrapeURI: ts.URL,
				ArtiSSLVerify: true,
				ArtiTimeout:   5 * time.Second,
				OptionalMetrics: config.OptionalMetrics{
					ReplicationStatus: tt.replicationStatus,
				},
				Credentials: &config.Credentials{
					AuthMethod: "userPass",
					Username:  "test",
					Password:  "pass",
				},
				Logger: slog.Default(),
			})

			got, err := client.FetchReplications()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchReplications() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.NodeId != tt.nodeId {
					t.Errorf("FetchReplications() nodeId = %v, want %v", got.NodeId, tt.nodeId)
				}
				if len(got.Replications) != len(tt.wantReplications) {
					t.Errorf("FetchReplications() got %v replications, want %v", len(got.Replications), len(tt.wantReplications))
					return
				}
				for i, replication := range got.Replications {
					if replication != tt.wantReplications[i] {
						t.Errorf("FetchReplications() replication[%d] = %v, want %v", i, replication, tt.wantReplications[i])
					}
				}
			}
		})
	}
} 