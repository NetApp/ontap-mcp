package main

import (
	"context"
	"crypto/tls"
	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/ontap"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

const NvmeCluster = "aff"
const NvmeClusterStr = "On the " + NvmeCluster + " cluster, "

func TestNVMe(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean NVMe subsystem",
			input:            NvmeClusterStr + "delete nvme subsystem " + rn("sys2") + " with in marketing svm with allow_delete_while_mapped and allow_delete_with_hosts",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=marketing&name=" + rn("sys2"), validationFunc: deleteObject},
		},
		{
			name:             "Create NVMe subsystem",
			input:            NvmeClusterStr + "create nvme subsystem " + rn("sys2") + " with linux os and with host nqns as nqn.1992-01.example.com:host1, nqn.1992-01.example.com:host2 on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=marketing&name=" + rn("sys2"), validationFunc: createObject},
		},
		{
			name:             "Update NVMe subsystem",
			input:            NvmeClusterStr + "add comment as `comment about the` in " + rn("sys2") + " nvme subsystem on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Add host in NVMe subsystem",
			input:            NvmeClusterStr + "add host nqn as nqn.1992-01.example.com:host3 in " + rn("sys2") + " nvme subsystem in marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Remove host in NVMe subsystem",
			input:            NvmeClusterStr + "remove host nqn as nqn.1992-01.example.com:host3 in " + rn("sys2") + " nvme subsystem in marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean NVMe subsystem",
			input:            NvmeClusterStr + "delete nvme subsystem " + rn("sys2") + " with in marketing svm with allow_delete_while_mapped and allow_delete_with_hosts",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=marketing&name=" + rn("sys2"), validationFunc: deleteObject},
		},
		{
			name:             "Clean NVMe subsystem",
			input:            NvmeClusterStr + "delete nvme subsystem " + rn("sys1") + " with in nvmevs1 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=nvmevs1&name=" + rn("sys1"), validationFunc: deleteObject},
		},
		{
			name:             "Create NVMe subsystem",
			input:            NvmeClusterStr + "create nvme subsystem " + rn("sys1") + " with linux os on the nvmevs1 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=nvmevs1&name=" + rn("sys1"), validationFunc: createObject},
		},
		{
			name:             "Clean NVMe namespace",
			input:            NvmeClusterStr + "delete nvme namespace '" + rn("/vol/docns/ns1") + "' with in nvmevs1 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/namespaces?svm.name=nvmevs1&name=" + rn(`/vol/docns/ns1`), validationFunc: deleteObject},
		},
		{
			name:             "Create NVMe namespace",
			input:            NvmeClusterStr + "create nvme namespace '" + rn("/vol/docns/ns1") + "' with linux os and 20mb size in nvmevs1 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/namespaces?svm.name=nvmevs1&name=" + rn(`/vol/docns/ns1`), validationFunc: createObject},
		},
		{
			name:             "Update NVMe namespace",
			input:            NvmeClusterStr + "update nvme namespace '" + rn("/vol/docns/ns1") + "' with to 40mb size in nvmevs1 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Create NVMe subsystem map",
			input:            NvmeClusterStr + "create subsystem map of " + rn("sys1") + " subsystem and '" + rn("/vol/docns/ns1") + "' namespace in nvmevs1 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystem-maps?svm.name=nvmevs1", validationFunc: verifySubsystemMaps(rn("sys1"), rn(`/vol/docns/ns1`), true)},
		},
		{
			name:             "Clean NVMe subsystem map",
			input:            NvmeClusterStr + "delete subsystem map of " + rn("sys1") + " subsystem and namespace '" + rn("/vol/docns/ns1") + "' in nvmevs1 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystem-maps?svm.name=nvmevs1", validationFunc: verifySubsystemMaps(rn("sys1"), rn(`/vol/docns/ns1`), false)},
		},
		{
			name:             "Clean NVMe namespace",
			input:            NvmeClusterStr + "delete nvme namespace '" + rn("/vol/docns/ns1") + "' with in nvmevs1 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/namespaces?svm.name=nvmevs1&name=" + rn(`/vol/docns/ns1`), validationFunc: deleteObject},
		},
		{
			name:             "Clean NVMe subsystem",
			input:            NvmeClusterStr + "delete nvme subsystem " + rn("sys1") + " with in nvmevs1 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nvme/subsystems?svm.name=nvmevs1&name=" + rn("sys1"), validationFunc: deleteObject},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	poller := cfg.Pollers[NvmeCluster]
	if poller == nil {
		t.Skipf("Cluster %q not found in %s, skipping NVMe tests", NvmeCluster, ConfigFile)
	}
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
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %s", tt.input)
			}
		})
	}
}

func verifySubsystemMaps(subsystemName, namespaceName string, exist bool) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool { //nolint:unparam
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type subsystemMapRecord struct {
			Namespace ontap.NameAndUUID `json:"namespace"`
			Subsystem ontap.NameAndUUID `json:"subsystem"`
		}
		type response struct {
			NumRecords int                  `json:"num_records"`
			Records    []subsystemMapRecord `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifySubsystemMaps: request failed: %v", err)
			return false
		}

		if exist {
			for _, record := range data.Records {
				gotSubsystem := record.Subsystem.Name
				gotNamespace := record.Namespace.Name
				if gotSubsystem == subsystemName && gotNamespace == namespaceName {
					return true
				}
			}
			t.Errorf("subsystem map does not exist")
		} else {
			sbsMapRecord := false
			for _, record := range data.Records {
				gotSubsystem := record.Subsystem.Name
				gotNamespace := record.Namespace.Name
				if gotSubsystem == subsystemName && gotNamespace == namespaceName {
					sbsMapRecord = true
					break
				}
			}
			if !sbsMapRecord {
				return true
			}
			t.Errorf("subsystem map exists")
		}
		return false
	}
}
