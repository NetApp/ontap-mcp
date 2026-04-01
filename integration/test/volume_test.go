package main

import (
	"context"
	"crypto/tls"
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
			name:             "List all volumes in one cluster in one svm with given fields",
			input:            ClusterStr + "for every volume on the marketing svm, show me the name, used size, available size, and snapshot policy",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?svm=marketing", validationFunc: listObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docs") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docsnew") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named " + rn("docs") + " on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "resize the " + rn("docs") + " volume on the marketing svm to 25MB",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "update junction path of the " + rn("docs") + " volume on the marketing svm to empty",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Enable volume autogrowth",
			input:            ClusterStr + "enable autogrowth and grow percent to 62 on the " + rn("docs") + " volume in the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename volume",
			input:            ClusterStr + "rename the " + rn("docs") + " volume on the marketing svm to " + rn("docsnew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the " + rn("docsnew") + " volume on the marketing svm to offline",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the " + rn("docsnew") + " volume on the marketing svm to online",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume junction path",
			input:            ClusterStr + "update junction path of the " + rn("docsnew") + " volume on the marketing svm to /" + rn("docsnew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "List one volume in one cluster in one svm with specific field",
			input:            ClusterStr + "for " + rn("docsnew") + " volume on the marketing svm, show me the name and junction path",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=marketing", validationFunc: listObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docs") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docsnew") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=marketing", validationFunc: deleteObject},
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
