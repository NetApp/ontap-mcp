package main

import (
	"context"
	"crypto/tls"
	"fmt"
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
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs")), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docsnew")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docsnew")), validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            fmt.Sprintf("%screate a 20MB volume named %s on the marketing svm and the harvest_vc_aggr aggregate", ClusterStr, rn("docs")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs")), validationFunc: createObject},
		},
		{
			name:             "Update volume size",
			input:            fmt.Sprintf("%sresize the %s volume on the marketing svm to 25MB", ClusterStr, rn("docs")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume size",
			input:            fmt.Sprintf("%supdate junction path of the %s volume on the marketing svm to empty", ClusterStr, rn("docs")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Enable volume autogrowth",
			input:            fmt.Sprintf("%senable autogrowth and grow percent to 62 on the %s volume in the marketing svm", ClusterStr, rn("docs")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename volume",
			input:            fmt.Sprintf("%srename the %s volume on the marketing svm to %s", ClusterStr, rn("docs"), rn("docsnew")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docsnew")), validationFunc: createObject},
		},
		{
			name:             "Update volume state",
			input:            fmt.Sprintf("%supdate state of the %s volume on the marketing svm to offline", ClusterStr, rn("docsnew")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume state",
			input:            fmt.Sprintf("%supdate state of the %s volume on the marketing svm to online", ClusterStr, rn("docsnew")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume junction path",
			input:            fmt.Sprintf("%supdate junction path of the %s volume on the marketing svm to /%s", ClusterStr, rn("docsnew"), rn("docsnew")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "List one volume in one cluster in one svm with specific field",
			input:            fmt.Sprintf("%sfor %s volume on the marketing svm, show me the name and junction path", ClusterStr, rn("docsnew")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docsnew")), validationFunc: listObject},
		},
		{
			name:             "Clean volume",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs")), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docsnew")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docsnew")), validationFunc: deleteObject},
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
