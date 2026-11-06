package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestValidateReplicationContract(t *testing.T) {
	const (
		sourceCluster      = "source"
		destinationCluster = "destination"
		sourceONTAPName    = "source-ontap"
		destinationONTAP   = "destination-ontap"
		sourceSVM          = "source-svm"
		destinationSVM     = "destination-svm"
		lifFields          = "fields=name%2Cenabled%2Cstate%2Cip.address%2Clocation.node.name&max_records=500&services=intercluster_core"
		clusterPeerFields  = "fields=name%2Cuuid%2Cstatus&max_records=500&name="
		svmPeerFields      = "fields=uuid%2Cstate%2Capplications%2Csvm.name%2Cpeer.cluster.name%2Cpeer.svm.name&max_records=500&peer.cluster.name="
	)
	clusterInfo := func(name string, major, minor int) exchange {
		return exchange{
			method: http.MethodGet,
			path:   "/api/cluster?fields=*",
			resp:   `{"name":"` + name + `","version":{"generation":9,"major":` + strconv.Itoa(major) + `,"minor":` + strconv.Itoa(minor) + `}}`,
		}
	}
	interclusterLIFs := exchange{
		method: http.MethodGet,
		path:   "/api/network/ip/interfaces?" + lifFields,
		resp:   `{"num_records":1,"records":[{"name":"icl1","enabled":true,"state":"up"}]}`,
	}
	clusterPeer := func(name string) exchange {
		return exchange{
			method: http.MethodGet,
			path:   "/api/cluster/peers?" + clusterPeerFields + name,
			resp:   `{"num_records":1,"records":[{"name":"` + name + `","uuid":"peer-uuid","status":{"state":"available"}}]}`,
		}
	}
	svmPeer := func(localSVM, peerCluster, peerSVM string) exchange {
		return exchange{
			method: http.MethodGet,
			path:   "/api/svm/peers?" + svmPeerFields + peerCluster + "&peer.svm.name=" + peerSVM + "&svm.name=" + localSVM,
			resp:   `{"num_records":1,"records":[{"uuid":"svm-peer-uuid","state":"peered","applications":["snapmirror"]}]}`,
		}
	}

	scripts := map[string][]exchange{
		sourceCluster: {
			interclusterLIFs,
			clusterInfo(sourceONTAPName, 16, 1),
			clusterPeer(destinationONTAP),
			svmPeer(sourceSVM, destinationONTAP, destinationSVM),
		},
		destinationCluster: {
			interclusterLIFs,
			clusterInfo(destinationONTAP, 15, 1),
			clusterPeer(sourceONTAPName),
			svmPeer(destinationSVM, sourceONTAPName, sourceSVM),
		},
	}
	cs := newMultiClusterSession(t, "multiplex", scripts)
	text, isErr := callTool(t, cs, "validate_replication", map[string]any{
		"source_cluster":      sourceCluster,
		"destination_cluster": destinationCluster,
		"source_svm":          sourceSVM,
		"destination_svm":     destinationSVM,
	})
	assert.False(t, isErr)

	var response ValidateReplicationResponse
	if err := json.Unmarshal([]byte(text), &response); err != nil {
		t.Fatalf("decoding tool output %q: %v", text, err)
	}
	assert.True(t, response.CanReplicate)
	assert.Equal(t, response.SourceVersion, "9.16.1")
	assert.Equal(t, response.DestinationVersion, "9.15.1")
	assert.Equal(t, len(response.Checks), 9)
	assert.Equal(t, len(response.MissingPrerequisites), 0)
	for _, check := range response.Checks {
		if !check.Passed || !check.Verifiable {
			t.Errorf("check %q was not satisfied: %#v", check.Name, check)
		}
	}
}

func TestValidateReplicationContractRejectsSameCluster(t *testing.T) {
	cs := newMultiClusterSession(t, "multiplex", map[string][]exchange{
		contractCluster: {},
	})
	text, isErr := callTool(t, cs, "validate_replication", map[string]any{
		"source_cluster":      contractCluster,
		"destination_cluster": contractCluster,
	})
	assert.True(t, isErr)
	assert.Contains(t, text, "must be different")
}
