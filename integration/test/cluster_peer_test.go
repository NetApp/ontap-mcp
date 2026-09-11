package main

import (
	"context"
	"crypto/tls"
	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/ontap"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestClusterPeer(t *testing.T) {
	SourceCluster := "umeng-aff300-05-06"
	SourceClusterStr := "On the " + SourceCluster + " cluster, "
	DestinationCluster := "aff"
	SkipIfMissing(t, CheckTools)

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	sourcePoller := cfg.Pollers[SourceCluster]
	sourceTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: sourcePoller.InsecureTLS(), // #nosec G402
		},
	}
	sourceClient := &http.Client{Transport: sourceTransport, Timeout: 10 * time.Second}

	destinationPoller := cfg.Pollers[DestinationCluster]
	destinationTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: destinationPoller.InsecureTLS(), // #nosec G402
		},
	}
	destinationClient := &http.Client{Transport: destinationTransport, Timeout: 10 * time.Second}

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
			verifyAPI:        ontapVerifier{api: "api/cluster/peers?fields=status.state&remote.name=", validationFunc: verifyClusterPeer(false, destinationPoller, destinationClient)},
		},
		{
			name:             "Create cluster peer",
			input:            SourceClusterStr + "create cluster peer relationship with " + DestinationCluster + " cluster",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/cluster/peers?fields=status.state&remote.name=", validationFunc: verifyClusterPeer(true, destinationPoller, destinationClient)},
		},
		{
			name:             "Remove cluster peer",
			input:            SourceClusterStr + "remove cluster peer relationship with " + DestinationCluster + " cluster",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/cluster/peers?fields=status.state&remote.name=", validationFunc: verifyClusterPeer(false, destinationPoller, destinationClient)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			if _, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr); err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, sourcePoller, sourceClient) {
				t.Errorf("Error while accessing the object via prompt %q", tt.input)
			}
		})
	}
}

func verifyClusterPeer(exist bool, destinationPoller *config.Poller, destinationClient *http.Client) func(t *testing.T, api string, sourcePoller *config.Poller, sourceClient *http.Client) bool {
	return func(t *testing.T, api string, sourcePoller *config.Poller, sourceClient *http.Client) bool {
		var (
			cl ontap.Cluster
		)
		params := url.Values{}
		params.Set("fields", "name")
		if err := requests.URL("https://"+destinationPoller.Addr+"/api/cluster").
			BasicAuth(destinationPoller.Username, destinationPoller.Password).
			Params(params).
			Client(destinationClient).
			ToJSON(&cl).
			Fetch(context.Background()); err != nil {
			t.Errorf("verifyClusterPeer: request failed: %v", err)
			return false
		}

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
		err := requests.URL("https://"+sourcePoller.Addr+"/"+api+cl.Name).
			BasicAuth(sourcePoller.Username, sourcePoller.Password).
			Client(sourceClient).
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
