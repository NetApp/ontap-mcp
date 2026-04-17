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

func TestIGroupLUNMap(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create SVM",
			input:            ClusterStr + "create " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Create iSCSI service",
			input:            ClusterStr + "create iscsi service on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Create svm scope network interface with ip 1",
			input:            ClusterStr + "create network interface named " + rn("svg1") + " in " + rn("marketing") + " svm with ip address 10.63.41.117 and netmask 18 on node umeng-aff300-05 with service policy as blocks",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=" + rn("svg1") + "&scope=svm", validationFunc: createObject},
		},
		{
			name:             "Create svm scope network interface with ip 2",
			input:            ClusterStr + "create network interface named " + rn("svg2") + " in " + rn("marketing") + " svm with ip address 10.63.41.118 and netmask 18 on node umeng-aff300-06 with service policy as blocks",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=" + rn("svg2") + "&scope=svm", validationFunc: createObject},
		},
		{
			name:             "Clean igroup igroupFin",
			input:            ClusterStr + "delete igroup " + rn("igroupFin") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFin") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean igroup igroupFinNew",
			input:            ClusterStr + "delete igroup " + rn("igroupFinNew") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFinNew") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume doc",
			input:            ClusterStr + "delete volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("doc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean LUN lundoc",
			input:            ClusterStr + "delete lun " + rn("lundoc") + " in volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundoc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create igroup",
			input:            ClusterStr + "create an igroup named " + rn("igroupFin") + " with OS type linux and protocol iscsi on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFin") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Add initiator to igroup",
			input:            ClusterStr + "add initiator iqn.2021-01.com.example:test to igroup " + rn("igroupFin") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFin") + "&svm.name=" + rn("marketing") + "&fields=initiators", validationFunc: verifyInitiator(true, "iqn.2021-01.com.example:test")},
		},
		{
			name:             "Remove initiator from igroup",
			input:            ClusterStr + "remove initiator iqn.2021-01.com.example:test from igroup " + rn("igroupFin") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFin") + "&svm.name=" + rn("marketing") + "&fields=initiators", validationFunc: verifyInitiator(false, "iqn.2021-01.com.example:test")},
		},
		{
			name:             "Rename igroup",
			input:            ClusterStr + "rename igroup from " + rn("igroupFin") + " to " + rn("igroupFinNew") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFinNew") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 100MB volume named " + rn("doc") + " on the " + rn("marketing") + " svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("doc") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Create LUN",
			input:            ClusterStr + "create a 20MB lun named " + rn("lundoc") + " in volume " + rn("doc") + " on the " + rn("marketing") + " svm with os type linux",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundoc") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Create lun map",
			input:            ClusterStr + "create lun map of lun named " + "/vol/" + rn("doc") + "/" + rn("lundoc") + " and an igroup named " + rn("igroupFinNew") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/lun-maps?igroup.name=" + rn("igroupFinNew") + "&lun.name=" + "/vol/" + rn("doc") + "/" + rn("lundoc") + "&svm.name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Clean lun map",
			input:            ClusterStr + "delete lun map of lun named " + "/vol/" + rn("doc") + "/" + rn("lundoc") + " and an igroup named " + rn("igroupFinNew") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/lun-maps?igroup.name=" + rn("igroupFinNew") + "&lun.name=" + "/vol/" + rn("doc") + "/" + rn("lundoc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("doc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean LUN",
			input:            ClusterStr + "delete lun " + rn("lundoc") + " in volume " + rn("doc") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/luns?name=/vol/" + rn("doc") + "/" + rn("lundoc") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean igroup",
			input:            ClusterStr + "delete igroup " + rn("igroupFinNew") + " on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/igroups?name=" + rn("igroupFinNew") + "&svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean svm scope network interface with ip 1",
			input:            ClusterStr + "delete svm scope network interface named " + rn("svg1") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=" + rn("svg1") + "&scope=svm", validationFunc: deleteObject},
		},
		{
			name:             "Clean svm scope network interface with ip 2",
			input:            ClusterStr + "delete svm scope network interface named " + rn("svg2") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/network/ip/interfaces?name=" + rn("svg2") + "&scope=svm", validationFunc: deleteObject},
		},
		{
			name:             "Update iSCSI service",
			input:            ClusterStr + "disabled iscsi service on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean iSCSI service",
			input:            ClusterStr + "delete iscsi service in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: deleteObject},
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

func verifyInitiator(exist bool, expectedInitiatorName string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type InitiatorName struct {
			Name string `json:"name"`
		}
		type IGroup struct {
			Initiators []InitiatorName `json:"initiators"`
		}
		type response struct {
			NumRecords int      `json:"num_records"`
			Records    []IGroup `json:"records"`
		}

		var data response
		var initiatorFound bool
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyInitiator: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyInitiator: expected 1 record, got %d", data.NumRecords)
			return false
		}
		gotIgroup := data.Records[0]
		for _, initiator := range gotIgroup.Initiators {
			if initiator.Name != expectedInitiatorName {
				continue
			}
			if !exist {
				t.Errorf("verifyInitiator: initiator should not be exist")
				return false
			}
			initiatorFound = true
		}
		if !initiatorFound && exist {
			t.Errorf("verifyInitiator: initiator must be exist")
			return false
		}
		return true
	}
}
