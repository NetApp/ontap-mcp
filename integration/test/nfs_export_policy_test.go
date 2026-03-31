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
			input:            fmt.Sprintf("%sdelete %s NFS export policy", ClusterStr, rn("nfsEngPolicy")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/nfs/export-policies?name=%s", rn("nfsEngPolicy")), validationFunc: deleteObject},
		},
		{
			name:             "Clean NFS export policy",
			input:            fmt.Sprintf("%sdelete %s NFS export policy", ClusterStr, rn("nfsMgrPolicy")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/nfs/export-policies?name=%s", rn("nfsMgrPolicy")), validationFunc: deleteObject},
		},
		{
			name:             "Create NFS export policy",
			input:            fmt.Sprintf("%screate an NFS export policy name %s on the marketing svm", ClusterStr, rn("nfsEngPolicy")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/nfs/export-policies?name=%s", rn("nfsEngPolicy")), validationFunc: createObject},
		},
		{
			name:             "Create volume",
			input:            fmt.Sprintf("%screate a 20MB volume named %s on the marketing svm and the harvest_vc_aggr aggregate", ClusterStr, rn("docs")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs")), validationFunc: createObject},
		},
		{
			name:             "Attach NFS export policy to volume",
			input:            fmt.Sprintf("%sapply %s NFS export policy to the %s volume in the marketing svm", ClusterStr, rn("nfsEngPolicy"), rn("docs")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename NFS export policy",
			input:            fmt.Sprintf("%srename the NFS export policy from %s to %s on the marketing svm", ClusterStr, rn("nfsEngPolicy"), rn("nfsMgrPolicy")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/nfs/export-policies?name=%s", rn("nfsMgrPolicy")), validationFunc: createObject},
		},
		{
			name:             "Clean volume",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs")), validationFunc: deleteObject},
		},
		{
			name:             "Clean NFS export policy",
			input:            fmt.Sprintf("%sdelete %s NFS export policy", ClusterStr, rn("nfsMgrPolicy")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/nfs/export-policies?name=%s", rn("nfsMgrPolicy")), validationFunc: deleteObject},
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
