package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

const (
	svmPeerSourceCluster      = "svm-peer-source"
	svmPeerDestinationCluster = "svm-peer-destination"
	svmPeerSourceSVM          = "source-svm"
	svmPeerDestinationSVM     = "destination-svm"
)

func svmPeerClusterInfo(name string) exchange {
	return exchange{method: http.MethodGet, path: "/api/cluster?fields=*", resp: `{"name":"` + name + `"}`}
}

func svmPeerLookup(localSVM, remoteSVM, remoteCluster, uuid, state string, applications []string) exchange {
	path := "/api/svm/peers?fields=uuid%2Cstate%2Capplications%2Csvm.name%2Cpeer.svm.name%2Cpeer.cluster.name"
	if remoteCluster != "" {
		path += "&peer.cluster.name=" + remoteCluster
	}
	path += "&peer.svm.name=" + remoteSVM + "&svm.name=" + localSVM

	e := exchange{method: http.MethodGet, path: path, resp: noRecords}
	if uuid != "" {
		quotedApplications := make([]string, len(applications))
		for i, application := range applications {
			quotedApplications[i] = strconv.Quote(application)
		}
		apps := "[" + strings.Join(quotedApplications, ",") + "]"
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `","state":"` + state + `","applications":` + apps + `}]}`
	}
	return e
}

func TestSVMPeerContract(t *testing.T) {
	peerArgs := map[string]any{
		"source_cluster_name":      svmPeerSourceCluster,
		"source_svm_name":          svmPeerSourceSVM,
		"destination_cluster_name": svmPeerDestinationCluster,
		"destination_svm_name":     svmPeerDestinationSVM,
	}

	tests := []struct {
		name    string
		tool    string
		args    map[string]any
		scripts map[string][]exchange
		want    string
		wantErr bool
	}{
		{
			name: "create proposes and accepts cross-cluster SVM peer",
			tool: "create_svm_peer",
			args: peerArgs,
			scripts: map[string][]exchange{
				svmPeerSourceCluster: {
					svmPeerClusterInfo("source-ontap"),
					svmPeerLookup(svmPeerSourceSVM, svmPeerDestinationSVM, "destination-ontap", "", "", nil),
					{method: http.MethodPost, path: "/api/svm/peers",
						body: `{"svm":{"name":"source-svm"},"peer":{"cluster":{"name":"destination-ontap"},"svm":{"name":"destination-svm"}},"applications":["snapmirror"]}`},
				},
				svmPeerDestinationCluster: {
					svmPeerClusterInfo("destination-ontap"),
					svmPeerLookup(svmPeerDestinationSVM, svmPeerSourceSVM, "source-ontap", "destination-peer-uuid", "pending", nil),
					{method: http.MethodPatch, path: "/api/svm/peers/destination-peer-uuid", body: `{"state":"peered"}`},
				},
			},
			want: "SVM peer created or updated successfully",
		},
		{
			name: "create same-cluster SVM peer",
			tool: "create_svm_peer",
			args: map[string]any{
				"source_cluster_name":      svmPeerSourceCluster,
				"source_svm_name":          svmPeerSourceSVM,
				"destination_cluster_name": svmPeerSourceCluster,
				"destination_svm_name":     svmPeerDestinationSVM,
			},
			scripts: map[string][]exchange{
				svmPeerSourceCluster: {
					svmPeerClusterInfo("source-ontap"),
					svmPeerLookup(svmPeerSourceSVM, svmPeerDestinationSVM, "source-ontap", "", "", nil),
					{method: http.MethodPost, path: "/api/svm/peers",
						body: `{"svm":{"name":"source-svm"},"peer":{"svm":{"name":"destination-svm"}},"applications":["snapmirror"]}`},
				},
			},
			want: "SVM peer created or updated successfully",
		},
		{
			name: "delete peer found on source cluster",
			tool: "delete_svm_peer",
			args: peerArgs,
			scripts: map[string][]exchange{
				svmPeerSourceCluster: {
					svmPeerClusterInfo("source-ontap"),
					svmPeerLookup(svmPeerSourceSVM, svmPeerDestinationSVM, "destination-ontap", "source-peer-uuid", "peered", []string{"snapmirror"}),
					{method: http.MethodDelete, path: "/api/svm/peers/source-peer-uuid"},
				},
				svmPeerDestinationCluster: {
					svmPeerClusterInfo("destination-ontap"),
				},
			},
			want: "SVM peer deleted successfully",
		},
		{
			name: "delete destination peer when source peer is missing",
			tool: "delete_svm_peer",
			args: peerArgs,
			scripts: map[string][]exchange{
				svmPeerSourceCluster: {
					svmPeerClusterInfo("source-ontap"),
					svmPeerLookup(svmPeerSourceSVM, svmPeerDestinationSVM, "destination-ontap", "", "", nil),
				},
				svmPeerDestinationCluster: {
					svmPeerClusterInfo("destination-ontap"),
					svmPeerLookup(svmPeerDestinationSVM, svmPeerSourceSVM, "source-ontap", "destination-peer-uuid", "peered", []string{"snapmirror"}),
					{method: http.MethodDelete, path: "/api/svm/peers/destination-peer-uuid"},
				},
			},
			want: "SVM peer deleted successfully",
		},
		{
			name: "delete reports when peer is missing from both clusters",
			tool: "delete_svm_peer",
			args: peerArgs,
			scripts: map[string][]exchange{
				svmPeerSourceCluster: {
					svmPeerClusterInfo("source-ontap"),
					svmPeerLookup(svmPeerSourceSVM, svmPeerDestinationSVM, "destination-ontap", "", "", nil),
				},
				svmPeerDestinationCluster: {
					svmPeerClusterInfo("destination-ontap"),
					svmPeerLookup(svmPeerDestinationSVM, svmPeerSourceSVM, "source-ontap", "", "", nil),
				},
			},
			want:    "SVM peer relationship not found",
			wantErr: true,
		},
		{
			name: "reject unsupported applications without contacting ONTAP",
			tool: "create_svm_peer",
			args: map[string]any{
				"source_cluster_name":      svmPeerSourceCluster,
				"source_svm_name":          svmPeerSourceSVM,
				"destination_cluster_name": svmPeerDestinationCluster,
				"destination_svm_name":     svmPeerDestinationSVM,
				"applications":             []string{"unsupported"},
			},
			scripts: map[string][]exchange{svmPeerSourceCluster: {}, svmPeerDestinationCluster: {}},
			want:    `application "unsupported" found`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newMultiClusterSession(t, "multiplex", tt.scripts)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			assert.Contains(t, text, tt.want)
		})
	}
}
