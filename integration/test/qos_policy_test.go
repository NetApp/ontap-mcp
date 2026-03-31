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
			input:            fmt.Sprintf("%sdelete %s QoS policy in marketing svm", ClusterStr, rn("gold")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("gold")), validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            fmt.Sprintf("%sdelete %s QoS policy in marketing svm", ClusterStr, rn("silver")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("silver")), validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            fmt.Sprintf("%sdelete %s QoS policy in marketing svm", ClusterStr, rn("payroll")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("payroll")), validationFunc: deleteObject},
		},
		{
			name:             "Create fixed QoS policy",
			input:            fmt.Sprintf("%screate a fixed QoS policy named %s on the marketing svm with a max throughput of 5000 iops", ClusterStr, rn("gold")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("gold")), validationFunc: createObject},
		},
		{
			name:             "Create adaptive QoS policy",
			input:            fmt.Sprintf("%screate a adaptive QoS policy named %s on the marketing svm with a expected iops as 2000 peak iops as 5000 and absolute min iops is 10", ClusterStr, rn("payroll")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("payroll")), validationFunc: createObject},
		},
		{
			name:             "Rename QoS policy",
			input:            fmt.Sprintf("%srename the QoS policy from %s to %s on the marketing svm", ClusterStr, rn("gold"), rn("silver")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("silver")), validationFunc: createObject},
		},
		{
			name:             "Clean QoS policy",
			input:            fmt.Sprintf("%sdelete %s QoS policy in marketing svm", ClusterStr, rn("silver")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("silver")), validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            fmt.Sprintf("%sdelete %s QoS policy in marketing svm", ClusterStr, rn("payroll")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("payroll")), validationFunc: deleteObject},
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
