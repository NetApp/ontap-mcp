package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"slices"
	"sort"
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
			expectedOntapErr: "entry doesn't exist",
			verifyAPI:        ontapVerifier{api: "api/name-services/dns?svm.name=" + rn("dnsSvc"), validationFunc: deleteObject},
		},
		{
			name:             "Create DNS",
			input:            ClusterStr + "create DNS configuration on the " + rn("dnsSvc") + " svm with domains example.com and nameservers 10.10.10.10, 10.10.10.30, 10.10.10.20 and disable dns config validation",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/name-services/dns?svm.name=" + rn("dnsSvc") + "&fields=domains,servers", validationFunc: verifyDNSConfig(1, []string{"example.com"}, []string{"10.10.10.30", "10.10.10.20", "10.10.10.10"})},
		},
		{
			name:             "Delete DNS",
			input:            ClusterStr + "delete DNS configuration in " + rn("dnsSvc") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/name-services/dns?svm.name=" + rn("dnsSvc"), validationFunc: verifyDNSConfig(0, []string{}, []string{})},
		},
		{
			name:             "Create DNS (validation enabled)",
			input:            ClusterStr + "create DNS configuration on the " + rn("dnsSvc") + " svm with domains example.com and nameservers 10.10.10.10",
			expectedOntapErr: "Verify that the network configuration is correct and that DNS servers are available.",
			verifyAPI:        ontapVerifier{api: "api/name-services/dns?svm.name=" + rn("dnsSvc") + "&fields=domains,servers", validationFunc: verifyDNSConfig(0, []string{}, []string{})},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("dnsSvc") + " svm",
			expectedOntapErr: "",
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
			InsecureSkipVerify: poller.InsecureTLS(), // #nosec G402
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

func verifyDNSConfig(expectedTotalRecords int, expectedDomains []string, expectedServers []string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
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

		if data.NumRecords != expectedTotalRecords {
			t.Errorf("verifyDNSConfig: expected %d record but got %d", expectedTotalRecords, data.NumRecords)
			return false
		}

		if len(data.Records) > 0 {
			rec := data.Records[0]
			expDomains := slices.Clone(expectedDomains)
			sort.Strings(expDomains)
			sort.Strings(rec.Domains)
			if len(rec.Domains) != len(expDomains) || !slices.Equal(rec.Domains, expDomains) {
				t.Errorf("verifyDNSConfig: expected domains %v but got %v", expDomains, rec.Domains)
				return false
			}
			expServers := slices.Clone(expectedServers)
			sort.Strings(expServers)
			sort.Strings(rec.Servers)
			if len(rec.Servers) != len(expServers) || !slices.Equal(rec.Servers, expServers) {
				t.Errorf("verifyDNSConfig: expected servers %v but got %v", expServers, rec.Servers)
				return false
			}
		}
		return true
	}
}
