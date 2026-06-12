package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/config"
)

func TestDNSService(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("dnsSvc") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("dnsSvc"), validationFunc: deleteObject},
		},
		{
			name:             "Create SVM",
			input:            ClusterStr + "create " + rn("dnsSvc") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("dnsSvc"), validationFunc: createObject},
		},
		{
			name:             "Clean DNS",
			input:            ClusterStr + "delete DNS configuration in " + rn("dnsSvc") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/name-services/dns?svm.name=" + rn("dnsSvc"), validationFunc: deleteObject},
		},
		{
			name:             "Create DNS",
			input:            ClusterStr + "create DNS configuration on the " + rn("dnsSvc") + " svm with domains example.com and nameservers 10.10.10.10",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/name-services/dns?svm.name=" + rn("dnsSvc") + "&fields=domains,servers", validationFunc: verifyDNSConfig},
		},
		{
			name:             "Delete DNS",
			input:            ClusterStr + "delete DNS configuration in " + rn("dnsSvc") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/name-services/dns?svm.name=" + rn("dnsSvc"), validationFunc: deleteObject},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("dnsSvc") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("dnsSvc"), validationFunc: deleteObject},
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

func verifyDNSConfig(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	type dnsRecord struct {
		Domains []string `json:"domains"`
		Servers []string `json:"servers"`
	}
	type response struct {
		NumRecords int         `json:"num_records"`
		Records    []dnsRecord `json:"records"`
	}

	var data response
	err := requests.URL("https://"+poller.Addr+"/"+api).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		ToJSON(&data).
		Fetch(context.Background())
	if err != nil {
		t.Errorf("verifyDNSConfig: request failed: %v", err)
		return false
	}

	if data.NumRecords != 1 {
		t.Errorf("verifyDNSConfig: expected 1 record, got %d", data.NumRecords)
		return false
	}

	rec := data.Records[0]
	if !slices.Contains(rec.Domains, "example.com") {
		t.Errorf("verifyDNSConfig: expected domains to contain 'example.com', got %v", rec.Domains)
		return false
	}
	if !slices.Contains(rec.Servers, "10.10.10.10") {
		t.Errorf("verifyDNSConfig: expected servers to contain '10.10.10.10', got %v", rec.Servers)
		return false
	}

	return true
}
