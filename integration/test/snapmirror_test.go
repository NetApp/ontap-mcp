package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/config"
)

func TestSnapMirror(t *testing.T) {
	SkipIfMissing(t, "CHECK_TOOLSS")

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean source SVM",
			input:            ClusterStr + "delete " + rn("srsvm") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("srsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Clean destination SVM",
			input:            ClusterStr + "delete " + rn("dtsvm") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("dtsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Create source SVM",
			input:            ClusterStr + "create " + rn("srsvm") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("srsvm"), validationFunc: createObject},
		},
		{
			name:             "Create destination SVM",
			input:            ClusterStr + "create " + rn("dtsvm") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("dtsvm"), validationFunc: createObject},
		},
		{
			name:             "Clean source volume",
			input:            ClusterStr + "delete volume " + rn("srvol") + " in " + rn("srsvm") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("srvol") + "&svm.name=" + rn("srsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Clean destination volume",
			input:            ClusterStr + "delete volume " + rn("dtvol") + " in " + rn("dtsvm") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("dtvol") + "&svm.name=" + rn("dtsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Create source volume",
			input:            ClusterStr + "create a 100MB volume named " + rn("srvol") + " on the " + rn("srsvm") + " svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("srvol") + "&svm.name=" + rn("srsvm"), validationFunc: createObject},
		},
		{
			name:             "Create destination volume",
			input:            ClusterStr + "create a 100MB volume named " + rn("dtvol") + " on the " + rn("dtsvm") + " svm and the harvest_vc_aggr aggregate with dp type",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("dtvol") + "&svm.name=" + rn("dtsvm"), validationFunc: createObject},
		},
		{
			name:             "Create SnapMirror relationship",
			input:            ClusterStr + "create a snapmirror relationship from source svm " + rn("srsvm") + " and source volume " + rn("srvol") + " to destination svm " + rn("dtsvm") + " and destination volume " + rn("dtvol") + " with policy name Asynchronous",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/snapmirror/relationships?destination.path=" + rn("dtsvm") + ":" + rn("dtvol") + "&fields=state,policy.name", validationFunc: verifySnapMirror(true, "Asynchronous", "uninitialized")},
		},
		{
			name:             "Update SnapMirror relationship",
			input:            ClusterStr + "update a snapmirror relationship of destination svm " + rn("dtsvm") + " and destination volume " + rn("dtvol") + " with transfer schedule name to hourly",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/snapmirror/relationships?destination.path=" + rn("dtsvm") + ":" + rn("dtvol") + "&fields=state,policy.name", validationFunc: verifySnapMirror(true, "Asynchronous", "uninitialized")},
		},
		{
			name:             "Update SnapMirror relationship 2",
			input:            ClusterStr + "update a snapmirror relationship of destination svm " + rn("dtsvm") + " and destination volume " + rn("dtvol") + " with policy name MirrorAndVault",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/snapmirror/relationships?destination.path=" + rn("dtsvm") + ":" + rn("dtvol") + "&fields=state,policy.name", validationFunc: verifySnapMirror(true, "MirrorAndVault", "uninitialized")},
		},
		{
			name:             "Initialize SnapMirror relationship",
			input:            ClusterStr + "initialize a snapmirror relationship of destination svm " + rn("dtsvm") + " and destination volume " + rn("dtvol"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/snapmirror/relationships?destination.path=" + rn("dtsvm") + ":" + rn("dtvol") + "&fields=state,policy.name", validationFunc: verifySnapMirror(true, "MirrorAndVault", "snapmirrored")},
		},
		{
			name:             "Break SnapMirror relationship",
			input:            ClusterStr + "break a snapmirror relationship of destination svm " + rn("dtsvm") + " and destination volume " + rn("dtvol"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/snapmirror/relationships?destination.path=" + rn("dtsvm") + ":" + rn("dtvol") + "&fields=state,policy.name", validationFunc: verifySnapMirror(true, "MirrorAndVault", "broken_off")},
		},
		{
			name:             "Resync SnapMirror relationship",
			input:            ClusterStr + "resync a snapmirror relationship of destination svm " + rn("dtsvm") + " and destination volume " + rn("dtvol"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/snapmirror/relationships?destination.path=" + rn("dtsvm") + ":" + rn("dtvol") + "&fields=state,policy.name", validationFunc: verifySnapMirror(true, "MirrorAndVault", "snapmirrored")},
		},
		{
			name:             "Delete SnapMirror relationship",
			input:            ClusterStr + "delete a snapmirror relationship of destination svm " + rn("dtsvm") + " and destination volume " + rn("dtvol"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/snapmirror/relationships?destination.path=" + rn("dtsvm") + ":" + rn("dtvol") + "&fields=state,policy.name", validationFunc: verifySnapMirror(false, "", "")},
		},
		{
			name:             "Clean source volume",
			input:            ClusterStr + "delete volume " + rn("srvol") + " in " + rn("srsvm") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("srvol") + "&svm.name=" + rn("srsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Clean destination volume",
			input:            ClusterStr + "delete volume " + rn("dtvol") + " in " + rn("dtsvm") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("dtvol") + "&svm.name=" + rn("dtsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Clean SVM peer",
			input:            ClusterStr + "delete svm peer of " + rn("srsvm") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/peers?name=" + rn("srsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Clean source SVM",
			input:            ClusterStr + "delete " + rn("srsvm") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("srsvm"), validationFunc: deleteObject},
		},
		{
			name:             "Clean destination SVM",
			input:            ClusterStr + "delete " + rn("dtsvm") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("dtsvm"), validationFunc: deleteObject},
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

func verifySnapMirror(exist bool, expectedPolicyName string, expectedState string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type NameData struct {
			Name string `json:"name,omitempty"`
		}
		type SnapMirrorRelationship struct {
			Policy NameData `json:"policy"`
			State  string   `json:"state"`
		}
		type response struct {
			NumRecords int                      `json:"num_records"`
			Records    []SnapMirrorRelationship `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifySnapMirror: request failed: %v", err)
			return false
		}

		if exist {
			if data.NumRecords == 0 {
				t.Errorf("verifySnapMirror: expected 1 record, got %d", data.NumRecords)
				return false
			}

			gotSnapMirror := data.Records[0]
			if gotSnapMirror.Policy.Name != expectedPolicyName {
				t.Errorf("verifySnapMirror: expected policy name %s, got %s", expectedPolicyName, gotSnapMirror.Policy.Name)
				return false
			}
			if gotSnapMirror.State != expectedState {
				t.Errorf("verifySnapMirror: expected state %s, got %s", expectedState, gotSnapMirror.State)
				return false
			}
			return true
		}

		if !exist && data.NumRecords > 0 {
			t.Errorf("verifySnapMirror: expected 0 record, got %d", data.NumRecords)
			return false
		}
		return true
	}
}
