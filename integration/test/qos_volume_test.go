package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/config"
)

func TestQoSVolumePolicy(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean QoS policy qos_docs_200iops",
			input:            fmt.Sprintf("%sdelete %s QoS policy in marketing svm", ClusterStr, rn("qos_docs_200iops")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("qos_docs_200iops")), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs_qos")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs_qos")), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos2",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs_qos2")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs_qos2")), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos3",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs_qos3")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs_qos3")), validationFunc: deleteObject},
		},

		{
			name:             "Create fixed QoS policy qos_docs_200iops",
			input:            fmt.Sprintf("%screate a fixed QoS policy named %s on the marketing svm with a max throughput of 200 iops and min throughput of 0 iops", ClusterStr, rn("qos_docs_200iops")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("qos_docs_200iops")), validationFunc: createObject},
		},
		{
			name:             "Create volume docs_qos",
			input:            fmt.Sprintf("%screate a 20MB volume named %s on the marketing svm and the harvest_vc_aggr aggregate", ClusterStr, rn("docs_qos")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs_qos")), validationFunc: createObject},
		},

		{
			name:             "Apply named QoS policy to existing volume",
			input:            fmt.Sprintf("%sapply the %s QoS policy to the %s volume on the marketing svm", ClusterStr, rn("qos_docs_200iops"), rn("docs_qos")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.name", rn("docs_qos")), validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},

		{
			name:             "Create volume docs_qos2 with named QoS policy",
			input:            fmt.Sprintf("%screate a 20MB volume named %s on the marketing svm and the harvest_vc_aggr aggregate with QoS policy %s", ClusterStr, rn("docs_qos2"), rn("qos_docs_200iops")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.name", rn("docs_qos2")), validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},

		{
			name:             "Create volume docs_qos3 with inline QoS max 300 iops",
			input:            fmt.Sprintf("%screate a 20MB volume named %s on the marketing svm and the harvest_vc_aggr aggregate with an inline QoS limit of max_iops 300", ClusterStr, rn("docs_qos3")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.max_throughput_iops", rn("docs_qos3")), validationFunc: verifyQoSMaxIOPS(300)},
		},
		{
			name:             "Update volume docs_qos3 inline QoS to max 150 iops",
			input:            fmt.Sprintf("%supdate the %s volume on the marketing svm setting an inline QoS limit of max_iops 150", ClusterStr, rn("docs_qos3")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.max_throughput_iops", rn("docs_qos3")), validationFunc: verifyQoSMaxIOPS(150)},
		},

		{
			name:             "Switch docs_qos3 from inline to named QoS policy",
			input:            fmt.Sprintf("%sapply the %s QoS policy to the %s volume on the marketing svm", ClusterStr, rn("qos_docs_200iops"), rn("docs_qos3")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.name", rn("docs_qos3")), validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},

		{
			name:             "Switch docs_qos from named policy to inline QoS",
			input:            fmt.Sprintf("%supdate the %s volume on the marketing svm setting an inline QoS limit of max_iops 100", ClusterStr, rn("docs_qos")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.max_throughput_iops", rn("docs_qos")), validationFunc: verifyQoSMaxIOPS(100)},
		},

		{
			name:             "Remove inline QoS from docs_qos by setting max_iops to 0",
			input:            fmt.Sprintf("%supdate the %s volume on the marketing svm and remove the inline QoS limit by setting max_iops to 0", ClusterStr, rn("docs_qos")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.name", rn("docs_qos")), validationFunc: verifyQoSNoPolicy},
		},

		{
			name:             "Re-apply named QoS policy to docs_qos3 before removal test",
			input:            fmt.Sprintf("%sapply the %s QoS policy to the %s volume on the marketing svm", ClusterStr, rn("qos_docs_200iops"), rn("docs_qos3")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.name", rn("docs_qos3")), validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},
		{
			name:             "Remove named QoS policy from docs_qos3 by specifying none",
			input:            fmt.Sprintf("%sremove the QoS policy from the %s volume on the marketing svm", ClusterStr, rn("docs_qos3")),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm.name=marketing&fields=qos.policy.name", rn("docs_qos3")), validationFunc: verifyQoSNoPolicy},
		},

		{
			name:             "Clean volume docs_qos after test",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs_qos")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs_qos")), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos2 after test",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs_qos2")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs_qos2")), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos3 after test",
			input:            fmt.Sprintf("%sdelete volume %s in marketing svm", ClusterStr, rn("docs_qos3")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/volumes?name=%s&svm=marketing", rn("docs_qos3")), validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy qos_docs_200iops after test",
			input:            fmt.Sprintf("%sdelete %s QoS policy in marketing svm", ClusterStr, rn("qos_docs_200iops")),
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: fmt.Sprintf("api/storage/qos/policies?name=%s", rn("qos_docs_200iops")), validationFunc: deleteObject},
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

func verifyQoSNoPolicy(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	type qosPolicy struct {
		Name           string `json:"name"`
		MaxThroughIOPS int    `json:"max_throughput_iops"`
	}
	type qos struct {
		Policy qosPolicy `json:"policy"`
	}
	type volumeRecord struct {
		QoS qos `json:"qos"`
	}
	type response struct {
		NumRecords int            `json:"num_records"`
		Records    []volumeRecord `json:"records"`
	}

	var data response
	err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		ToJSON(&data).
		Fetch(context.Background())
	if err != nil {
		t.Errorf("verifyQoSNoPolicy: request failed: %v", err)
		return false
	}
	if data.NumRecords != 1 {
		t.Errorf("verifyQoSNoPolicy: expected 1 record, got %d", data.NumRecords)
		return false
	}
	pol := data.Records[0].QoS.Policy
	if pol.Name != "" || pol.MaxThroughIOPS != 0 {
		t.Errorf("verifyQoSNoPolicy: expected no QoS policy, got name=%q max_throughput_iops=%d", pol.Name, pol.MaxThroughIOPS)
		return false
	}
	return true
}

func verifyQoSAssigned(policyName string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool { //nolint:unparam
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type qosPolicy struct {
			Name string `json:"name"`
		}
		type qos struct {
			Policy qosPolicy `json:"policy"`
		}
		type volumeRecord struct {
			QoS qos `json:"qos"`
		}
		type response struct {
			NumRecords int            `json:"num_records"`
			Records    []volumeRecord `json:"records"`
		}

		var data response
		err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyQoSAssigned: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyQoSAssigned: expected 1 record, got %d", data.NumRecords)
			return false
		}
		got := data.Records[0].QoS.Policy.Name
		if got != policyName {
			t.Errorf("verifyQoSAssigned: qos.policy.name = %q, want %q", got, policyName)
			return false
		}
		return true
	}
}

func verifyQoSMaxIOPS(wantIOPS int) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type qosPolicy struct {
			MaxThroughIOPS int `json:"max_throughput_iops"`
		}
		type qos struct {
			Policy qosPolicy `json:"policy"`
		}
		type volumeRecord struct {
			QoS qos `json:"qos"`
		}
		type response struct {
			NumRecords int            `json:"num_records"`
			Records    []volumeRecord `json:"records"`
		}

		var data response
		err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyQoSMaxIOPS: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyQoSMaxIOPS: expected 1 record, got %d", data.NumRecords)
			return false
		}
		got := data.Records[0].QoS.Policy.MaxThroughIOPS
		if got != wantIOPS {
			t.Errorf("verifyQoSMaxIOPS: qos.policy.max_throughput_iops = %d, want %d", got, wantIOPS)
			return false
		}
		return true
	}
}
