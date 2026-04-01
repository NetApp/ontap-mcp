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

const SarCluster = "sar"
const SarClusterStr = "On the " + SarCluster + " cluster, "

func TestNVMeService(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Update NVMe service",
			input:            SarClusterStr + "update nvme service to disable on the marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean NVMe service",
			input:            SarClusterStr + "delete nvme service in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/services?svm.name=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Create NVMe service",
			input:            SarClusterStr + "create nvme service on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/services?svm.name=marketing", validationFunc: createObject},
		},
		{
			name:             "Create NVMe subsystem",
			input:            SarClusterStr + "create nvme subsystem sys1 with linux os on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=marketing&name=sys1", validationFunc: createObject},
		},
		{
			name:             "Create NVMe subsystem",
			input:            SarClusterStr + "create nvme subsystem sys2 with linux os and with host nqns as nqn.1992-01.example.com:host1, nqn.1992-01.example.com:host2 on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=marketing&name=sys2", validationFunc: createObject},
		},
		{
			name:             "Update NVMe subsystem",
			input:            SarClusterStr + "add comment as `comment about the` in sys1 nvme subsystem linux os on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Add host in NVMe subsystem",
			input:            SarClusterStr + "add host nqn as nqn.1992-01.example.com:host1 in sys1 nvme subsystem linux os in marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Remove host in NVMe subsystem",
			input:            SarClusterStr + "remove host nqn as nqn.1992-01.example.com:host1 in sys1 nvme subsystem linux os in marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean NVMe subsystem",
			input:            SarClusterStr + "delete nvme subsystem sys1 with linux os in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=marketing&name=sys1", validationFunc: deleteObject},
		},
		{
			name:             "Clean NVMe subsystem",
			input:            SarClusterStr + "delete nvme subsystem sys2 with linux os in marketing svm with allow_delete_while_mapped and allow_delete_with_hosts",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=marketing&name=sys2", validationFunc: deleteObject},
		},
		{
			name:             "Update NVMe service",
			input:            SarClusterStr + "update nvme service to disable on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean NVMe service",
			input:            SarClusterStr + "delete nvme service in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/services?svm.name=marketing", validationFunc: deleteObject},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	poller := cfg.Pollers[SarCluster]
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
