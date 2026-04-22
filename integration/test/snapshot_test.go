package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/ontap"

	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/config"
)

func TestSnapshot(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create SVM",
			input:            ClusterStr + "create " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docs") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named " + rn("docs") + " on the " + rn("marketing") + " svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Clean snapshot",
			input:            ClusterStr + "Delete " + rn("localsnap") + " snapshot in " + rn("docs") + " volume in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: verifySnapshot("")},
		},
		{
			name:             "Create snapshot",
			input:            ClusterStr + "create a snapshot named " + rn("localsnap") + " in the volume " + rn("docs") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: verifySnapshot("")},
		},
		{
			name:             "Restore volume from snapshot",
			input:            ClusterStr + "Restore " + rn("doc") + " volume from a snapshot named " + rn("localsnap") + " in the " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: verifySnapshot("")},
		},
		{
			name:             "Clean snapshot",
			input:            ClusterStr + "Delete " + rn("localsnap") + " snapshot in " + rn("docs") + " volume in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: verifySnapshot("")},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docs") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: deleteObject},
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

func verifySnapshot(snapshotName string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool { //nolint:unparam
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		var data ontap.GetData
		err := requests.URL("https://"+poller.Addr+"/"+api).
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
