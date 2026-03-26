package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/config"
)

func TestQtree(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean qtree staff",
			input:            ClusterStr + "delete staff qtree in doc volume in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qtrees?name=staff", validationFunc: deleteObject},
		},
		{
			name:             "Clean qtree pay",
			input:            ClusterStr + "Delete pay qtree in docs volume in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qtrees?name=pay", validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Create qtree staff",
			input:            ClusterStr + "create a qtree named staff in docs volume on the marketing SVM",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qtrees?name=staff", validationFunc: verifyQtreeName("staff")},
		},
		{
			name:             "Rename qtree staff",
			input:            ClusterStr + "rename a qtree named staff to pay in docs volume on the marketing SVM",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qtrees?name=pay", validationFunc: verifyQtreeName("pay")},
		},
		{
			name:             "Clean qtree policy pay",
			input:            ClusterStr + "Delete pay qtree in docs volume in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qtrees?name=pay", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume docs in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: deleteObject},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	poller := cfg.Pollers[Cluster]
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: poller.UseInsecureTLS, // #nosec G402
		},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			if _, err = testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr); err != nil {
				slog.Error("Error processing input", slog.Any("error", err))
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %s", tt.input)
			}
		})
	}
}

func verifyQtreeName(qtreeName string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool { //nolint:unparam
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		var data ontap.GetData
		err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyQtreeName: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyQtreeName: expected 1 record, got %d", data.NumRecords)
			return false
		}
		got := data.Records[0].Name
		if got != qtreeName {
			t.Errorf("verifyQtreeName: qtree.name = %q, want %q", got, qtreeName)
			return false
		}
		return true
	}
}
