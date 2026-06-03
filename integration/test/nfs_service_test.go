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

func TestNFSService(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("nfsSvc") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("nfsSvc"), validationFunc: deleteObject},
		},
		{
			name:             "Create SVM",
			input:            ClusterStr + "create " + rn("nfsSvc") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("nfsSvc"), validationFunc: createObject},
		},
		{
			name:             "Clean NFS service",
			input:            ClusterStr + "delete NFS service in " + rn("nfsSvc") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/services?svm.name=" + rn("nfsSvc"), validationFunc: deleteObject},
		},
		{
			name:             "Create NFS service",
			input:            ClusterStr + "create NFS service on the " + rn("nfsSvc") + " svm with NFSv3 enabled and NFSv4.0 disabled",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/services?svm.name=" + rn("nfsSvc"), validationFunc: createObject},
		},
		{
			name:             "Update NFS service",
			input:            ClusterStr + "update NFS service on the " + rn("nfsSvc") + " svm to enable NFSv4.0",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Delete NFS service",
			input:            ClusterStr + "delete NFS service in " + rn("nfsSvc") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/services?svm.name=" + rn("nfsSvc"), validationFunc: deleteObject},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("nfsSvc") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("nfsSvc"), validationFunc: deleteObject},
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
