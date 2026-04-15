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

func TestSVM(t *testing.T) {
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
			name:             "Rename SVM",
			input:            ClusterStr + "rename svm " + rn("marketing") + " to " + rn("marketingNew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketingNew"), validationFunc: createObject},
		},
		{
			name:             "Update SVM",
			input:            ClusterStr + "update svm " + rn("marketingNew") + " state to stopped and comment as `stop_svm`",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketingNew") + "&fields=state,comment", validationFunc: verifySVM("stopped", "stop_svm")},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketingNew") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketingNew"), validationFunc: deleteObject},
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
			if _, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr); err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %q", tt.input)
			}
		})
	}
}

func verifySVM(expectedState string, expectedComment string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type SVM struct {
			Name    string `json:"name"`
			State   string `json:"state"`
			Comment string `json:"comment"`
		}
		type response struct {
			NumRecords int   `json:"num_records"`
			Records    []SVM `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifySVM: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifySVM: expected 1 record, got %d", data.NumRecords)
			return false
		}

		gotSVM := data.Records[0]
		if gotSVM.State != expectedState {
			t.Errorf("verifySVM: got state = %s, want %s", gotSVM.State, expectedState)
			return false
		}
		if gotSVM.Comment != expectedComment {
			t.Errorf("verifySVM: got comment = %s, want %s", gotSVM.Comment, expectedComment)
			return false
		}
		return true
	}
}
