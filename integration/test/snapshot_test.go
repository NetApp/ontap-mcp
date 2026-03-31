package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestSnapshot(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
		mustContain      []string
	}{
		{
			name:             "Clean snapshot policy every4hours",
			input:            fmt.Sprintf("%sdelete %s snapshot policy in marketing svm", ClusterStr, rn("every4hours")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/snapshot-policies?name=%s", rn("every4hours")), validationFunc: deleteObject},
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            fmt.Sprintf("%sDelete %s snapshot policy in marketing svm", ClusterStr, rn("every5min")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/snapshot-policies?name=%s", rn("every5min")), validationFunc: deleteObject},
		},
		{
			name:             "Create snapshot policy every4hours",
			input:            fmt.Sprintf("%screate a snapshot policy named %s on the marketing SVM. The schedule is 4hours and keeps the last 5 snapshots", ClusterStr, rn("every4hours")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/snapshot-policies?name=%s", rn("every4hours")), validationFunc: createObject},
		},
		{
			name:             "Create snapshot policy every5min",
			input:            fmt.Sprintf("%screate a snapshot policy named %s on the marketing SVM. The schedule is 5minutes and keeps the last 2 snapshots", ClusterStr, rn("every5min")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/snapshot-policies?name=%s", rn("every5min")), validationFunc: createObject},
		},
		{
			name:             "Clean snapshot policy every4hours",
			input:            fmt.Sprintf("%sdelete %s snapshot policy in marketing svm", ClusterStr, rn("every4hours")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/snapshot-policies?name=%s", rn("every4hours")), validationFunc: deleteObject},
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            fmt.Sprintf("%sDelete %s snapshot policy in marketing svm", ClusterStr, rn("every5min")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/snapshot-policies?name=%s", rn("every5min")), validationFunc: deleteObject},
		},
		{
			name:        "List snapshots on a volume",
			input:       ClusterStr + "list snapshots on volume harvest_root on svm harvest",
			mustContain: []string{"snapshot"},
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
			response, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr)
			if err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %q", tt.input)
			}
			lower := strings.ToLower(response)
			for _, want := range tt.mustContain {
				if !strings.Contains(lower, strings.ToLower(want)) {
					t.Errorf("response missing expected text %q \nfull response: %s", want, response)
				}
			}
		})
	}
}
