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

const (
	SVMPeerVsimCluster  = "vsim"
	SVMPeerUmengCluster = "umeng-aff300-05-06"
)

type svmPeerVerification struct {
	cluster  string
	verifier ontapVerifier
}

func TestSVMPeer(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	localSourceSVM := rn("peer_local_src")
	localDestinationSVM := rn("peer_local_dst")
	//nolint:gocritic
	//remoteSourceSVM := rn("peer_remote_src")
	//remoteDestinationSVM := rn("peer_remote_dst")

	localPeerAPI := "api/svm/peers?svm.name=" + localSourceSVM + "&peer.svm.name=" + localDestinationSVM + "&fields=state,applications"
	//nolint:gocritic
	//remoteSourcePeerAPI := "api/svm/peers?svm.name=" + remoteSourceSVM + "&peer.svm.name=" + remoteDestinationSVM + "&fields=state,applications"
	//remoteDestinationPeerAPI := "api/svm/peers?svm.name=" + remoteDestinationSVM + "&peer.svm.name=" + remoteSourceSVM + "&fields=state,applications"

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifications    []svmPeerVerification
	}{
		{
			name:  "Delete local source SVM on vsim",
			input: "On the " + SVMPeerVsimCluster + " cluster, Delete the " + localSourceSVM + " SVM",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: "api/svm/svms?name=" + localSourceSVM, validationFunc: deleteObject},
			}},
		},
		{
			name:  "Delete local destination SVM on vsim",
			input: "On the " + SVMPeerVsimCluster + " cluster, Delete the " + localDestinationSVM + " SVM",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: "api/svm/svms?name=" + localDestinationSVM, validationFunc: deleteObject},
			}},
		},
		{
			name:  "Create local source SVM on vsim",
			input: "On the " + SVMPeerVsimCluster + " cluster, create " + localSourceSVM + " svm",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: "api/svm/svms?name=" + localSourceSVM, validationFunc: createObject},
			}},
		},
		{
			name:  "Create local destination SVM on vsim",
			input: "On the " + SVMPeerVsimCluster + " cluster, create " + localDestinationSVM + " svm",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: "api/svm/svms?name=" + localDestinationSVM, validationFunc: createObject},
			}},
		},
		{
			name:  "Create local SVM peer on vsim",
			input: "Use create_svm_peer with application snapmirror from the " + localSourceSVM + " SVM on the " + SVMPeerVsimCluster + " cluster to the " + localDestinationSVM + " SVM on the " + SVMPeerVsimCluster + " cluster",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: localPeerAPI, validationFunc: verifySVMPeer},
			}},
		},
		{
			name:  "Delete local SVM peer on vsim",
			input: "Use delete_svm_peer to delete the SVM peer relationship from the " + localSourceSVM + " SVM on the " + SVMPeerVsimCluster + " cluster to the " + localDestinationSVM + " SVM on the " + SVMPeerVsimCluster + " cluster",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: localPeerAPI, validationFunc: deleteObject},
			}},
		},
		{
			name:  "Delete local source SVM on vsim",
			input: "Use modify_svm with operation delete for the " + localSourceSVM + " SVM on the " + SVMPeerVsimCluster + " cluster",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: "api/svm/svms?name=" + localSourceSVM, validationFunc: deleteObject},
			}},
		},
		{
			name:  "Delete local destination SVM on vsim",
			input: "Use modify_svm with operation delete for the " + localDestinationSVM + " SVM on the " + SVMPeerVsimCluster + " cluster",
			verifications: []svmPeerVerification{{
				cluster:  SVMPeerVsimCluster,
				verifier: ontapVerifier{api: "api/svm/svms?name=" + localDestinationSVM, validationFunc: deleteObject},
			}},
		},
		//nolint:gocritic
		//{
		//	name:  "Delete remote source SVM on vsim",
		//	input: "On the " + SVMPeerVsimCluster + " cluster, Delete the " + remoteSourceSVM + " SVM",
		//	verifications: []svmPeerVerification{{
		//		cluster:  SVMPeerVsimCluster,
		//		verifier: ontapVerifier{api: "api/svm/svms?name=" + remoteSourceSVM, validationFunc: deleteObject},
		//	}},
		//},
		//{
		//	name:  "Delete remote destination SVM on umeng",
		//	input: "On the " + SVMPeerVsimCluster + " cluster, Delete the " + remoteDestinationSVM + " SVM",
		//	verifications: []svmPeerVerification{{
		//		cluster:  SVMPeerUmengCluster,
		//		verifier: ontapVerifier{api: "api/svm/svms?name=" + remoteDestinationSVM, validationFunc: deleteObject},
		//	}},
		//},
		//{
		//	name:  "Create remote source SVM on vsim",
		//	input: "On the " + SVMPeerVsimCluster + " cluster, create " + remoteSourceSVM + " svm",
		//	verifications: []svmPeerVerification{{
		//		cluster:  SVMPeerVsimCluster,
		//		verifier: ontapVerifier{api: "api/svm/svms?name=" + remoteSourceSVM, validationFunc: createObject},
		//	}},
		//},
		//{
		//	name:  "Create remote destination SVM on umeng",
		//	input: "On the " + SVMPeerUmengCluster + " cluster, create " + remoteDestinationSVM + " svm",
		//	verifications: []svmPeerVerification{{
		//		cluster:  SVMPeerUmengCluster,
		//		verifier: ontapVerifier{api: "api/svm/svms?name=" + remoteDestinationSVM, validationFunc: createObject},
		//	}},
		//},
		// Create cluster peer
		//{
		//	name:  "Create remote SVM peer from vsim to umeng",
		//	input: "Use create_svm_peer with application snapmirror from the " + remoteSourceSVM + " SVM on the " + SVMPeerVsimCluster + " cluster to the " + remoteDestinationSVM + " SVM on the " + SVMPeerUmengCluster + " cluster",
		//	verifications: []svmPeerVerification{
		//		{cluster: SVMPeerVsimCluster, verifier: ontapVerifier{api: remoteSourcePeerAPI, validationFunc: verifySVMPeer}},
		//		{cluster: SVMPeerUmengCluster, verifier: ontapVerifier{api: remoteDestinationPeerAPI, validationFunc: verifySVMPeer}},
		//	},
		//},
		//{
		//	name:  "Delete remote SVM peer from vsim to umeng",
		//	input: "Use delete_svm_peer to delete the SVM peer relationship from the " + remoteSourceSVM + " SVM on the " + SVMPeerVsimCluster + " cluster to the " + remoteDestinationSVM + " SVM on the " + SVMPeerUmengCluster + " cluster",
		//	verifications: []svmPeerVerification{
		//		{cluster: SVMPeerVsimCluster, verifier: ontapVerifier{api: remoteSourcePeerAPI, validationFunc: deleteObject}},
		//		{cluster: SVMPeerUmengCluster, verifier: ontapVerifier{api: remoteDestinationPeerAPI, validationFunc: deleteObject}},
		//	},
		//},
		//{
		//	name:  "Delete remote source SVM on vsim",
		//	input: "Use modify_svm with operation delete for the " + remoteSourceSVM + " SVM on the " + SVMPeerVsimCluster + " cluster",
		//	verifications: []svmPeerVerification{{
		//		cluster:  SVMPeerVsimCluster,
		//		verifier: ontapVerifier{api: "api/svm/svms?name=" + remoteSourceSVM, validationFunc: deleteObject},
		//	}},
		//},
		//{
		//	name:  "Delete remote destination SVM on umeng",
		//	input: "Use modify_svm with operation delete for the " + remoteDestinationSVM + " SVM on the " + SVMPeerUmengCluster + " cluster",
		//	verifications: []svmPeerVerification{{
		//		cluster:  SVMPeerUmengCluster,
		//		verifier: ontapVerifier{api: "api/svm/svms?name=" + remoteDestinationSVM, validationFunc: deleteObject},
		//	}},
		//},
		// Delete cluster peer
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	clients := make(map[string]*http.Client, 2)
	for _, cluster := range []string{SVMPeerVsimCluster, SVMPeerUmengCluster} {
		poller := cfg.Pollers[cluster]
		if poller == nil {
			t.Skipf("Cluster %q not found in %s, skipping SVM peer tests", cluster, ConfigFile)
		}
		clients[cluster] = &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: poller.InsecureTLS()}}, // #nosec G402
			Timeout:   10 * time.Second,
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			if _, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr); err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			for _, verification := range tt.verifications {
				poller := cfg.Pollers[verification.cluster]
				verifier := verification.verifier
				if !verifier.validationFunc(t, verifier.api, poller, clients[verification.cluster]) {
					t.Errorf("Error verifying prompt %q on cluster %q", tt.input, verification.cluster)
				}
			}
		})
	}
}

func verifySVMPeer(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	t.Helper()

	type svmPeer struct {
		Applications []string `json:"applications"`
		State        string   `json:"state"`
	}
	type response struct {
		NumRecords int       `json:"num_records"`
		Records    []svmPeer `json:"records"`
	}

	var data response
	err := requests.URL("https://"+poller.Addr+"/"+api).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		ToJSON(&data).
		Fetch(context.Background())
	if err != nil {
		t.Errorf("verifySVMPeer: request failed: %v", err)
		return false
	}
	if data.NumRecords != 1 {
		t.Errorf("verifySVMPeer: expected 1 record, got %d", data.NumRecords)
		return false
	}
	if data.Records[0].State != "peered" {
		t.Errorf("verifySVMPeer: expected state peered, got %s", data.Records[0].State)
		return false
	}
	if !slices.Contains(data.Records[0].Applications, "snapmirror") {
		t.Errorf("verifySVMPeer: expected snapmirror application, got %v", data.Records[0].Applications)
		return false
	}
	return true
}
