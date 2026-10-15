package server

import (
	"net/http"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

const (
	peerSource      = "source-cluster"
	peerDestination = "destination-cluster"
	noRecords       = `{"num_records":0,"records":[]}`
)

func interclusterLIFs(resp string) exchange {
	return exchange{method: http.MethodGet, path: "/api/network/ip/interfaces?enabled=true&fields=ip.address&services=intercluster_core&state=up", resp: resp}
}

// createPeer expects a peer POST naming the remote LIFs. Both sides must send the same passphrase.
func createPeer(remoteIPs string, status int, resp string) exchange {
	return exchange{method: http.MethodPost, path: "/api/cluster/peers", status: status, resp: resp,
		body: `{"remote":{"ip_addresses":` + remoteIPs + `},"authentication":{"passphrase":"{{passphrase}}"}}`}
}

func clusterName(name string) exchange {
	return exchange{method: http.MethodGet, path: "/api/cluster?fields=name", resp: `{"name":"` + name + `"}`}
}

func lookupPeer(remoteName, uuid string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/cluster/peers?fields=uuid&remote.name=" + remoteName, resp: noRecords}
	if uuid != "" {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

func deletePeer(uuid string) exchange {
	return exchange{method: http.MethodDelete, path: "/api/cluster/peers/" + uuid}
}

func TestClusterPeerContract(t *testing.T) {
	const (
		sourceLIFs = `{"num_records":2,"records":[{"ip":{"address":"10.0.0.1"}},{"ip":{"address":"10.0.0.2"}}]}`
		destLIFs   = `{"num_records":1,"records":[{"ip":{"address":"10.1.0.1"}}]}`
		sourcePeer = `{"num_records":1,"records":[{"uuid":"src-peer-uuid"}]}`
		destPeer   = `{"num_records":1,"records":[{"uuid":"dst-peer-uuid"}]}`
		sourceIPs  = `["10.0.0.1","10.0.0.2"]`
		destIPs    = `["10.1.0.1"]`
	)
	peerArgs := map[string]any{"source_cluster_name": peerSource, "destination_cluster_name": peerDestination}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		source   []exchange
		dest     []exchange
		wantText string
		alsoWant []string // further substrings the result must contain
		wantErr  bool
	}{
		{
			name: "create offers destination LIFs on source and approves with source LIFs",
			tool: "create_cluster_peer",
			args: peerArgs,
			source: []exchange{
				interclusterLIFs(sourceLIFs),
				createPeer(destIPs, http.StatusCreated, sourcePeer),
			},
			dest: []exchange{
				interclusterLIFs(destLIFs),
				createPeer(sourceIPs, http.StatusCreated, destPeer),
			},
			wantText: "cluster peer relationship created successfully",
		},
		{
			// The recent CI failure: the shared source cluster had no intercluster LIFs up.
			name:     "create fails cleanly when source has no intercluster LIFs",
			tool:     "create_cluster_peer",
			args:     peerArgs,
			source:   []exchange{interclusterLIFs(noRecords)},
			wantText: "failed to find up and running intercluster LIFS for cluster=" + peerSource,
			wantErr:  true,
		},
		{
			name:     "create fails before touching source when destination has no intercluster LIFs",
			tool:     "create_cluster_peer",
			args:     peerArgs,
			source:   []exchange{interclusterLIFs(sourceLIFs)},
			dest:     []exchange{interclusterLIFs(noRecords)},
			wantText: "failed to find up and running intercluster LIFS for cluster=" + peerDestination,
			wantErr:  true,
		},
		{
			name: "create surfaces ONTAP error from LIF lookup",
			tool: "create_cluster_peer",
			args: peerArgs,
			source: []exchange{
				{method: http.MethodGet, path: "/api/network/ip/interfaces?enabled=true&fields=ip.address&services=intercluster_core&state=up",
					status: http.StatusForbidden, resp: `{"error":{"message":"not authorized for that command","code":"6"}}`},
			},
			wantText: "not authorized for that command",
			wantErr:  true,
		},
		{
			name: "create rolls back source peer when destination approval fails",
			tool: "create_cluster_peer",
			args: peerArgs,
			source: []exchange{
				interclusterLIFs(sourceLIFs),
				createPeer(destIPs, http.StatusCreated, sourcePeer),
				deletePeer("src-peer-uuid"),
			},
			dest: []exchange{
				interclusterLIFs(destLIFs),
				createPeer(sourceIPs, http.StatusBadRequest, `{"error":{"message":"Cluster peer authentication failed","code":"4653060"}}`),
			},
			wantText: "Cluster peer authentication failed",
			wantErr:  true,
		},
		{
			// Both errors must survive: the approval failure explains why, the rollback failure
			// tells the user a half-created peer offer is left on the source cluster.
			name: "create reports approval and rollback errors when rollback fails",
			tool: "create_cluster_peer",
			args: peerArgs,
			source: []exchange{
				interclusterLIFs(sourceLIFs),
				createPeer(destIPs, http.StatusCreated, sourcePeer),
				{method: http.MethodDelete, path: "/api/cluster/peers/src-peer-uuid", status: http.StatusInternalServerError,
					resp: `{"error":{"message":"rollback exploded","code":"1"}}`},
			},
			dest: []exchange{
				interclusterLIFs(destLIFs),
				createPeer(sourceIPs, http.StatusBadRequest, `{"error":{"message":"Cluster peer authentication failed","code":"4653060"}}`),
			},
			wantText: "Cluster peer authentication failed",
			alsoWant: []string{"rollback of source cluster peer src-peer-uuid failed, remove it manually", "rollback exploded"},
			wantErr:  true,
		},
		{
			name: "create rejects source POST response without a peer record",
			tool: "create_cluster_peer",
			args: peerArgs,
			source: []exchange{
				interclusterLIFs(sourceLIFs),
				createPeer(destIPs, http.StatusCreated, noRecords),
			},
			dest:     []exchange{interclusterLIFs(destLIFs)},
			wantText: "cluster peer not found",
			wantErr:  true,
		},
		{
			name:     "create with unknown destination cluster sends nothing",
			tool:     "create_cluster_peer",
			args:     map[string]any{"source_cluster_name": peerSource, "destination_cluster_name": "nope"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},
		{
			// Peering a cluster with itself would trip the per-cluster lock and look retryable.
			name:     "create rejects the same source and destination cluster",
			tool:     "create_cluster_peer",
			args:     map[string]any{"source_cluster_name": peerSource, "destination_cluster_name": peerSource},
			wantText: "source and destination clusters must differ",
			wantErr:  true,
		},
		{
			name: "delete removes the peer from both sides",
			tool: "delete_cluster_peer",
			args: peerArgs,
			source: []exchange{
				clusterName("src-name"),
				lookupPeer("dst-name", "src-side-uuid"),
				deletePeer("src-side-uuid"),
			},
			dest: []exchange{
				lookupPeer("src-name", "dst-side-uuid"),
				clusterName("dst-name"),
				deletePeer("dst-side-uuid"),
			},
			wantText: "cluster peer relationship deleted successfully",
		},
		{
			name:     "delete rejects the same source and destination cluster",
			tool:     "delete_cluster_peer",
			args:     map[string]any{"source_cluster_name": peerDestination, "destination_cluster_name": peerDestination},
			wantText: "source and destination clusters must differ",
			wantErr:  true,
		},
		{
			// The integration suite's first "Remove cluster peer" step depends on this message.
			name:     "delete when no peer exists",
			tool:     "delete_cluster_peer",
			args:     peerArgs,
			source:   []exchange{clusterName("src-name")},
			dest:     []exchange{lookupPeer("src-name", "")},
			wantText: "failed to find cluster peer relationships for remote cluster=src-name because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete still removes destination side when source delete fails",
			tool: "delete_cluster_peer",
			args: peerArgs,
			source: []exchange{
				clusterName("src-name"),
				lookupPeer("dst-name", "src-side-uuid"),
				{method: http.MethodDelete, path: "/api/cluster/peers/src-side-uuid", status: http.StatusBadRequest,
					resp: `{"error":{"message":"Cannot delete peer with existing SnapMirror relationships","code":"4653061"}}`},
			},
			dest: []exchange{
				lookupPeer("src-name", "dst-side-uuid"),
				clusterName("dst-name"),
				deletePeer("dst-side-uuid"),
			},
			wantText: "Cannot delete peer with existing SnapMirror relationships",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newMultiClusterSession(t, "multiplex", map[string][]exchange{peerSource: tt.source, peerDestination: tt.dest})
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			assert.Contains(t, text, tt.wantText)
			for _, want := range tt.alsoWant {
				assert.Contains(t, text, want)
			}
		})
	}
}
