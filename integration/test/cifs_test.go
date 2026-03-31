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

func TestCIFSShare(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean CIFS share",
			input:            fmt.Sprintf("%sdelete %s CIFS share in vs_test4 svm", ClusterStr, rn("cifsFin")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/cifs/shares?name=%s", rn("cifsFin")), validationFunc: deleteObject},
		},
		{
			name:             "Create CIFS share",
			input:            fmt.Sprintf("%screate CIFS share named %s with path as / on the vs_test4 svm", ClusterStr, rn("cifsFin")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/cifs/shares?name=%s", rn("cifsFin")), validationFunc: createObject},
		},
		{
			name:             "Update CIFS share",
			input:            fmt.Sprintf("%supdate CIFS share %s path to /vol_test2 on the vs_test4 svm", ClusterStr, rn("cifsFin")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean CIFS share",
			input:            fmt.Sprintf("%sdelete %s CIFS share in vs_test4 svm", ClusterStr, rn("cifsFin")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/protocols/cifs/shares?name=%s", rn("cifsFin")), validationFunc: deleteObject},
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
