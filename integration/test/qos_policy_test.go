package main

import (
	"context"
	"crypto/tls"
	"github.com/carlmjohnson/requests"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestQoSPolicy(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete " + rn("gold") + " QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("gold"), validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete " + rn("silver") + " QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("silver"), validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete " + rn("payroll") + " QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("payroll"), validationFunc: deleteObject},
		},
		{
			name:             "Create fixed QoS policy",
			input:            ClusterStr + "create a fixed QoS policy named " + rn("gold") + " on the marketing svm with a max throughput of 5000 iops",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("gold"), validationFunc: createObject},
		},
		{
			name:             "Create adaptive QoS policy",
			input:            ClusterStr + "create a adaptive QoS policy named " + rn("payroll") + " on the marketing svm with a expected iops as 2000 peak iops as 5000 and absolute min iops is 10",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("payroll"), validationFunc: createObject},
		},
		{
			name:             "Rename QoS policy",
			input:            ClusterStr + "rename the QoS policy from " + rn("gold") + " to " + rn("silver") + " on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("silver"), validationFunc: createObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete " + rn("silver") + " QoS policy in marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("silver"), validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete " + rn("payroll") + " QoS policy in marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("payroll"), validationFunc: deleteObject},
		},
		{
			name:             "Clean adaptive allocation QoS policy",
			input:            ClusterStr + "delete " + rn("alloc") + " QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("alloc"), validationFunc: deleteObject},
		},
		//nolint:gocritic These tests are available 9.10 onwards, current cluster is at 9.9, so skipping these tests in CI for now
		{
			name:             "Create adaptive QoS policy with allocation mode",
			input:            ClusterStr + "create an adaptive QoS policy named " + rn("alloc") + " on the marketing svm with expected iops 1000 peak iops 3000 absolute min iops 50 expected iops allocation allocated_space peak iops allocation used_space block size any",
			expectedOntapErr: "Unexpected argument",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("alloc") + "&fields=adaptive", validationFunc: verifyQoSAdaptiveFields(true, "allocated_space", "used_space", "any")},
		},
		{
			name:             "Update adaptive QoS policy allocation mode only",
			input:            ClusterStr + "update the " + rn("alloc") + " QoS policy on the marketing svm to use allocated_space for both expected and peak IOPS allocation",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("alloc") + "&fields=adaptive", validationFunc: verifyQoSAdaptiveFields(true, "allocated_space", "allocated_space", "any")},
		},
		{
			name:             "Clean adaptive allocation QoS policy",
			input:            ClusterStr + "delete " + rn("alloc") + " QoS policy in marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("alloc"), validationFunc: deleteObject},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	poller := cfg.Pollers[Cluster]
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: poller.InsecureTLS(), // #nosec G402
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

// verifyQoSAdaptiveFields returns a verifier that GETs the QoS policy with
// adaptive fields and asserts that the expected_iops_allocation,
// peak_iops_allocation, and block_size values match.
func verifyQoSAdaptiveFields(skipTesting bool, expectedIOPSAlloc, peakIOPSAlloc, blockSize string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		if skipTesting {
			t.Log("The cluster do not support this operation")
			return true
		}
		type adaptiveFields struct {
			ExpectedIOPSAllocation string `json:"expected_iops_allocation"`
			PeakIOPSAllocation     string `json:"peak_iops_allocation"`
			BlockSize              string `json:"block_size"`
		}
		type qosRecord struct {
			Adaptive adaptiveFields `json:"adaptive"`
		}
		type response struct {
			NumRecords int         `json:"num_records"`
			Records    []qosRecord `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyQoSAdaptiveFields: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyQoSAdaptiveFields: expected 1 record but got %d", data.NumRecords)
			return false
		}
		got := data.Records[0].Adaptive
		if got.ExpectedIOPSAllocation != expectedIOPSAlloc {
			t.Errorf("verifyQoSAdaptiveFields: expected_iops_allocation: want %q, got %q", expectedIOPSAlloc, got.ExpectedIOPSAllocation)
			return false
		}
		if got.PeakIOPSAllocation != peakIOPSAlloc {
			t.Errorf("verifyQoSAdaptiveFields: peak_iops_allocation: want %q, got %q", peakIOPSAlloc, got.PeakIOPSAllocation)
			return false
		}
		if got.BlockSize != blockSize {
			t.Errorf("verifyQoSAdaptiveFields: block_size: want %q, got %q", blockSize, got.BlockSize)
			return false
		}
		return true
	}
}
