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

func TestIscsiProtocol(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean iSCSI service",
			input:            ClusterStr + "delete iscsi service in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Create iSCSI service",
			input:            ClusterStr + "create iscsi service target named alias tgpath on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=marketing", validationFunc: createObject},
		},
		{
			name:             "Clean cluster scope network interface",
			input:            ClusterStr + "delete cluster scope network interface named cl_mg",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=cl_mg&scope=cluster", validationFunc: deleteObject},
		},
		{
			name:             "Clean svm scope network interface",
			input:            ClusterStr + "delete svm scope network interface named svg1 in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=svg1&scope=svm", validationFunc: deleteObject},
		},
		{
			name:             "Create cluster scope network interface",
			input:            ClusterStr + "create network interface named cl_mg with ip address 10.63.41.6 and netmask 18 with Default ipspace on node umeng-aff300-06",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=cl_mg&scope=cluster", validationFunc: createObject},
		},
		{
			name:             "Create svm scope network interface with ip",
			input:            ClusterStr + "create network interface named svg1 in marketing svm with ip address 10.63.41.7 and netmask 18 on node umeng-aff300-06",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=svg1&scope=svm", validationFunc: createObject},
		},
		{
			name:             "Update network interface",
			input:            ClusterStr + "change auto revert to false in cluster scoped network interface named cl_mg",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean cluster scope network interface",
			input:            ClusterStr + "delete cluster scope network interface named cl_mg",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=cl_mg&scope=cluster", validationFunc: deleteObject},
		},
		{
			name:             "Clean svm scope network interface",
			input:            ClusterStr + "delete svm scope network interface named svg1 in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=svg1&scope=svm", validationFunc: deleteObject},
		},
		{
			name:             "Create svm scope network interface with broadcast domain",
			input:            ClusterStr + "create network interface named svg1 in marketing svm with ip address 10.63.41.7 and netmask 18 on broadcast domain as Default",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=svg1&scope=svm", validationFunc: createObject},
		},
		{
			name:             "Clean svm scope network interface",
			input:            ClusterStr + "delete svm scope network interface named svg1 in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=svg1&scope=svm", validationFunc: deleteObject},
		},
		{
			name:             "Update iSCSI service",
			input:            ClusterStr + "disabled iscsi service on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean iSCSI service",
			input:            ClusterStr + "delete iscsi service in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=marketing", validationFunc: deleteObject},
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
