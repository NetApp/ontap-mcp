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

func TestVolume(t *testing.T) {
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
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docsnew") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named " + rn("docs") + " on the " + rn("marketing") + " svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "resize the " + rn("docs") + " volume on the " + rn("marketing") + " svm to 25MB",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "update junction path of the " + rn("docs") + " volume on the " + rn("marketing") + " svm to empty",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Enable volume autogrowth",
			input:            ClusterStr + "enable autogrowth and grow percent to 62 on the " + rn("docs") + " volume in the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename volume",
			input:            ClusterStr + "rename the " + rn("docs") + " volume on the " + rn("marketing") + " svm to " + rn("docsnew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm to offline",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm to online",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume junction path",
			input:            ClusterStr + "update junction path of the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm to /" + rn("docsnew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume files maximum",
			input:            ClusterStr + "increase maximum number of files to 2000 on the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			// Maximum number of files would not be increase exactly same as requested, the discrepancy happens due to how ONTAP calculates and allocates internal file structures (inodes).
			verifyAPI: ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing") + "&fields=files.maximum", validationFunc: verifyFilesMax(2000)},
		},
		{
			name:             "Create thick-provisioned volume",
			input:            ClusterStr + "create a 50MB thick-provisioned volume named " + rn("thick") + " on the " + rn("marketing") + " svm and the harvest_vc_aggr aggregate with space guarantee type volume and snapshot reserve 5 percent",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("thick") + "&svm=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Update volume snapshot policy",
			input:            ClusterStr + "set the snapshot policy of the " + rn("thick") + " volume on the " + rn("marketing") + " svm to none",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume efficiency",
			input:            ClusterStr + "disable compression and deduplication on the " + rn("thick") + " volume on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean thick volume",
			input:            ClusterStr + "delete volume " + rn("thick") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("thick") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docs") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docsnew") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketing") + " svm",
			expectedOntapErr: "",
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

func verifyFilesMax(expectedFilesMax int) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		possibleVariation := 10
		type Files struct {
			Maximum *int `json:"maximum,omitzero"`
		}
		type Volume struct {
			Files Files `json:"files,omitzero"`
		}
		type response struct {
			NumRecords int      `json:"num_records"`
			Records    []Volume `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyFilesMax: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyFilesMax: expected 1 record, got %d", data.NumRecords)
			return false
		}

		gotVolume := data.Records[0]
		if gotVolume.Files.Maximum == nil {
			t.Errorf("verifyFilesMax: nil files.maximum found")
			return false
		}

		if v := *gotVolume.Files.Maximum; v < (expectedFilesMax-possibleVariation) || v >= expectedFilesMax {
			t.Errorf("verifyFilesMax: files.maximum value is not in range %d - %d, got %d", expectedFilesMax-possibleVariation, expectedFilesMax, v)
			return false
		}
		return true
	}
}
