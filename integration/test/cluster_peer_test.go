package main

import (
	"context"
	"crypto/tls"
	"github.com/carlmjohnson/requests"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestClusterPeer(t *testing.T) {
	SourceCluster := "aff"
	SourceClusterStr := "On the " + SourceCluster + " cluster, "
	DestinationCluster := "umeng-aff300-05-06"
	SkipIfMissing(t, CheckTools)
	tests := []struct {
		name             string
		input            string
		afxInput         string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Remove cluster peer",
			input:            SourceClusterStr + "remove cluster peer relationship with " + DestinationCluster + " cluster",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/cluster/peers?remote.name=" + DestinationCluster + "&fields=status.state", validationFunc: verifyClusterPeer(false)},
		},
		{
			name:             "Create cluster peer",
			input:            SourceClusterStr + "create cluster peer relationship with " + DestinationCluster + " cluster",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/cluster/peers?remote.name=" + DestinationCluster + "&fields=status.state", validationFunc: verifyClusterPeer(true)},
		},
		{
			name:             "Remove cluster peer",
			input:            SourceClusterStr + "remove cluster peer relationship with " + DestinationCluster + " cluster",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/cluster/peers?remote.name=" + DestinationCluster + "&fields=status.state", validationFunc: verifyClusterPeer(false)},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	poller := cfg.Pollers[SourceCluster]
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: poller.InsecureTLS(), // #nosec G402
		},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			if _, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr); err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %q", tt.input)
			}
		})
	}
}

func verifyClusterPeer(exist bool) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		// Cluster requires some time to reach the state value to available for cluster peer operation
		time.Sleep(10 * time.Second)
		type Status struct {
			State string `json:"state"`
		}
		type ClusterPeer struct {
			Status Status `json:"status"`
		}
		type response struct {
			NumRecords int           `json:"num_records"`
			Records    []ClusterPeer `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyClusterPeer: request failed: %v", err)
			return false
		}

		if exist {
			if data.NumRecords != 1 {
				t.Errorf("verifyClusterPeer: expected 1 record, got %d", data.NumRecords)
				return false
			}

			gotClusterPeer := data.Records[0]
			if gotClusterPeer.Status.State != "available" {
				t.Errorf("verifyClusterPeer: got state = %s, want %s", gotClusterPeer.Status.State, "available")
				return false
			}
		} else if data.NumRecords > 0 {
			t.Errorf("verifyClusterPeer: expected 0 record, got %d", data.NumRecords)
			return false
		}
		return true
	}
}
