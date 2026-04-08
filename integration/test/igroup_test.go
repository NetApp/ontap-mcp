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

func TestIGroup(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean igroup",
			input:            ClusterStr + "delete igroup " + rn("igroupFin") + " on the vs_test4 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFin") + "&svm.name=vs_test4", validationFunc: deleteObject},
		},
		{
			name:             "Create igroup",
			input:            ClusterStr + "create an igroup named " + rn("igroupFin") + " with OS type linux and protocol iscsi on the vs_test4 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFin") + "&svm.name=vs_test4", validationFunc: createObject},
		},
		{
			name:             "Update igroup",
			input:            ClusterStr + "rename igroup " + rn("igroupFin") + " to " + rn("igroupFinNew") + " on the vs_test4 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFinNew") + "&svm.name=vs_test4", validationFunc: createObject},
		},
		{
			name:             "Add initiator to igroup",
			input:            ClusterStr + "add initiator iqn.2021-01.com.example:test to igroup " + rn("igroupFinNew") + " on the vs_test4 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Remove initiator from igroup",
			input:            ClusterStr + "remove initiator iqn.2021-01.com.example:test from igroup " + rn("igroupFinNew") + " on the vs_test4 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean igroup",
			input:            ClusterStr + "delete igroup " + rn("igroupFinNew") + " on the vs_test4 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFinNew") + "&svm.name=vs_test4", validationFunc: deleteObject},
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
