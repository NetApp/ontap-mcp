package main

import (
	"context"
	"crypto/tls"
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
			input:            ClusterStr + "delete " + rn("qos_docs_200iops") + " QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("qos_docs_200iops"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos",
			input:            ClusterStr + "delete volume " + rn("docs_qos") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos2",
			input:            ClusterStr + "delete volume " + rn("docs_qos2") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos2") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos3",
			input:            ClusterStr + "delete volume " + rn("docs_qos3") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos3") + "&svm=marketing", validationFunc: deleteObject},
		},

		{
			name:             "Create fixed QoS policy qos_docs_200iops",
			input:            ClusterStr + "create a fixed QoS policy named " + rn("qos_docs_200iops") + " on the marketing svm with a max throughput of 200 iops and min throughput of 0 iops",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("qos_docs_200iops"), validationFunc: createObject},
		},
		{
			name:             "Create volume docs_qos",
			input:            ClusterStr + "create a 20MB volume named " + rn("docs_qos") + " on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos") + "&svm=marketing", validationFunc: createObject},
		},

		{
			name:             "Apply named QoS policy to existing volume",
			input:            ClusterStr + "apply the " + rn("qos_docs_200iops") + " QoS policy to the " + rn("docs_qos") + " volume on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos") + "&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},

		{
			name:             "Create volume docs_qos2 with named QoS policy",
			input:            ClusterStr + "create a 20MB volume named " + rn("docs_qos2") + " on the marketing svm and the harvest_vc_aggr aggregate with QoS policy " + rn("qos_docs_200iops"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos2") + "&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},

		{
			name:             "Create volume docs_qos3 with inline QoS max 300 iops",
			input:            ClusterStr + "create a 20MB volume named " + rn("docs_qos3") + " on the marketing svm and the harvest_vc_aggr aggregate with an inline QoS limit of max_iops 300",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos3") + "&svm.name=marketing&fields=qos.policy.max_throughput_iops", validationFunc: verifyQoSMaxIOPS(300)},
		},
		{
			name:             "Update volume docs_qos3 inline QoS to max 150 iops",
			input:            ClusterStr + "update the " + rn("docs_qos3") + " volume on the marketing svm setting an inline QoS limit of max_iops 150",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos3") + "&svm.name=marketing&fields=qos.policy.max_throughput_iops", validationFunc: verifyQoSMaxIOPS(150)},
		},

		{
			name:             "Switch docs_qos3 from inline to named QoS policy",
			input:            ClusterStr + "apply the " + rn("qos_docs_200iops") + " QoS policy to the " + rn("docs_qos3") + " volume on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos3") + "&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},

		{
			name:             "Switch docs_qos from named policy to inline QoS",
			input:            ClusterStr + "update the " + rn("docs_qos") + " volume on the marketing svm setting an inline QoS limit of max_iops 100",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos") + "&svm.name=marketing&fields=qos.policy.max_throughput_iops", validationFunc: verifyQoSMaxIOPS(100)},
		},

		{
			name:             "Remove inline QoS from docs_qos by setting max_iops to 0",
			input:            ClusterStr + "update the " + rn("docs_qos") + " volume on the marketing svm and remove the inline QoS limit by setting max_iops to 0",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos") + "&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSNoPolicy},
		},

		{
			name:             "Re-apply named QoS policy to docs_qos3 before removal test",
			input:            ClusterStr + "apply the " + rn("qos_docs_200iops") + " QoS policy to the " + rn("docs_qos3") + " volume on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos3") + "&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSAssigned(rn("qos_docs_200iops"))},
		},
		{
			name:             "Remove named QoS policy from docs_qos3 by specifying none",
			input:            ClusterStr + "remove the QoS policy from the " + rn("docs_qos3") + " volume on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos3") + "&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSNoPolicy},
		},

		{
			name:             "Clean volume docs_qos after test",
			input:            ClusterStr + "delete volume " + rn("docs_qos") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos2 after test",
			input:            ClusterStr + "delete volume " + rn("docs_qos2") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos2") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos3 after test",
			input:            ClusterStr + "delete volume " + rn("docs_qos3") + " in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs_qos3") + "&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy qos_docs_200iops after test",
			input:            ClusterStr + "delete " + rn("qos_docs_200iops") + " QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=" + rn("qos_docs_200iops"), validationFunc: deleteObject},
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
	err := requests.URL("https://"+poller.Addr+"/"+api).
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
		err := requests.URL("https://"+poller.Addr+"/"+api).
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
		err := requests.URL("https://"+poller.Addr+"/"+api).
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
