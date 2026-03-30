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

func TestNFSExportPolicy(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean NFS export policy",
			input:            ClusterStr + "delete nfsEngPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsEngPolicy", validationFunc: deleteObject},
		},
		{
			name:             "Clean NFS export policy",
			input:            ClusterStr + "delete nfsMgrPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsMgrPolicy", validationFunc: deleteObject},
		},
		{
			name:             "Create NFS export policy",
			input:            ClusterStr + "create an NFS export policy name nfsEngPolicy on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsEngPolicy", validationFunc: createObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Attach NFS export policy to volume",
			input:            ClusterStr + "apply nfsEngPolicy NFS export policy to the docs volume in the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename NFS export policy",
			input:            ClusterStr + "rename the NFS export policy from nfsEngPolicy to nfsMgrPolicy on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsMgrPolicy", validationFunc: createObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume docs in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean NFS export policy",
			input:            ClusterStr + "delete nfsMgrPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsMgrPolicy", validationFunc: deleteObject},
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
