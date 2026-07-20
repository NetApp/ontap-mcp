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

func TestLUN(t *testing.T) {
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
			name:             "Clean LUN",
			input:            ClusterStr + "delete lun " + rn("lundoc") + " in volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundoc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean LUN",
			input:            ClusterStr + "delete lun " + rn("lundocnew") + " in volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundocnew") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("doc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 100MB volume named " + rn("doc") + " on the " + rn("marketing") + " svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("doc") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Create LUN",
			input:            ClusterStr + "create a 20MB lun named " + rn("lundoc") + " in volume " + rn("doc") + " on the " + rn("marketing") + " svm with os type linux",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundoc") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Create thick-provisioned LUN",
			input:            ClusterStr + "create a 10MB thick-provisioned lun named " + rn("lundocthick") + " with space guarantee in volume " + rn("doc") + " on the " + rn("marketing") + " svm with os type linux",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundocthick") + "&svm.name=" + rn("marketing") + "&fields=space.guarantee", validationFunc: verifyLUNSpaceGuarantee(true)},
		},
		{
			name:             "Clean thick LUN",
			input:            ClusterStr + "delete lun " + rn("lundocthick") + " in volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundocthick") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Update lun size",
			input:            ClusterStr + "update lun " + rn("lundoc") + " size to 50mb in volume " + rn("doc") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename lun",
			input:            ClusterStr + "rename the lun " + rn("lundoc") + " in volume " + rn("doc") + " on the " + rn("marketing") + " svm to " + rn("lundocnew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundocnew") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Update lun state",
			input:            ClusterStr + "disable the lun " + rn("lundocnew") + " in volume " + rn("doc") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean LUN",
			input:            ClusterStr + "delete lun " + rn("lundocnew") + " in volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundocnew") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("doc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
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

// verifyLUNSpaceGuarantee returns a verifier that GETs the LUN with
// space.guarantee fields and asserts that space.guarantee.requested
// matches the expected value.
func verifyLUNSpaceGuarantee(wantRequested bool) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type spaceGuarantee struct {
			Requested bool `json:"requested"`
		}
		type lunSpace struct {
			Guarantee spaceGuarantee `json:"guarantee"`
		}
		type lunRecord struct {
			Space lunSpace `json:"space"`
		}
		type response struct {
			NumRecords int         `json:"num_records"`
			Records    []lunRecord `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyLUNSpaceGuarantee: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyLUNSpaceGuarantee: expected 1 record but got %d", data.NumRecords)
			return false
		}
		got := data.Records[0].Space.Guarantee.Requested
		if got != wantRequested {
			t.Errorf("verifyLUNSpaceGuarantee: space.guarantee.requested: want %v, got %v", wantRequested, got)
			return false
		}
		return true
	}
}
