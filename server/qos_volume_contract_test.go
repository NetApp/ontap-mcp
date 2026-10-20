package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestQoSVolumeContract(t *testing.T) {
	const (
		volumesPath = "/api/storage/volumes"
		accepted    = `{"job":{"uuid":"qos-vol-job"}}`
		cdot        = `{"name":"contract-cluster","uuid":"c-uuid","version":{"full":"NetApp Release 9.16.1","generation":9,"major":16,"minor":1},"san_optimized":false,"disaggregated":false}`
	)
	cluster := exchange{method: http.MethodGet, path: "/api/cluster?fields=*", resp: cdot}
	jobDone := exchange{method: http.MethodGet, path: "/api/cluster/jobs/qos-vol-job", resp: `{"uuid":"qos-vol-job","state":"success"}`}

	// lookup is the name-to-UUID lookup that precedes every volume update; an empty uuid means
	// the volume does not exist.
	lookup := func(volume, uuid string) exchange {
		e := exchange{method: http.MethodGet, path: volumesPath + "?fields=uuid&svm.name=marketing&name=" + volume, resp: `{"num_records":0,"records":[]}`}
		if uuid != "" {
			e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
		}
		return e
	}
	patchQoS := func(policy string) exchange {
		return exchange{method: http.MethodPatch, path: volumesPath + "/vol-uuid", body: `{"qos":{"policy":` + policy + `}}`, status: http.StatusAccepted, resp: accepted}
	}
	createWithQoS := func(policy string) exchange {
		return exchange{method: http.MethodPost, path: volumesPath, status: http.StatusAccepted, resp: accepted,
			body: `{"svm":{"name":"marketing"},"name":"docs_qos2","aggregates":[{"name":"harvest_vc_aggr"}],"size":20971520,"qos":{"policy":` + policy + `}}`}
	}
	createArgs := func(qos map[string]any) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs_qos2", "aggregate_name": "harvest_vc_aggr", "size": "20MB", "qos": qos}
	}
	updateArgs := func(volume string, qos map[string]any) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": volume, "volume_update": map[string]any{"qos": qos}}
	}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			name:     "apply named policy to existing volume",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos", map[string]any{"policy_name": "qos_docs_200iops"}),
			ontap:    []exchange{lookup("docs_qos", "vol-uuid"), patchQoS(`{"name":"qos_docs_200iops"}`), jobDone},
			wantText: "Volume updated successfully",
		},
		{
			name:     "create volume with named policy",
			tool:     "create_volume",
			args:     createArgs(map[string]any{"policy_name": "qos_docs_200iops"}),
			ontap:    []exchange{cluster, createWithQoS(`{"name":"qos_docs_200iops"}`), jobDone},
			wantText: "Volume created successfully",
		},
		{
			name:     "create volume with inline max iops",
			tool:     "create_volume",
			args:     createArgs(map[string]any{"max_iops": "300"}),
			ontap:    []exchange{cluster, createWithQoS(`{"max_throughput_iops":300}`), jobDone},
			wantText: "Volume created successfully",
		},
		{
			name:     "create volume with every inline limit",
			tool:     "create_volume",
			args:     createArgs(map[string]any{"max_iops": "300", "min_iops": "10", "max_mbps": "50", "min_mbps": "5"}),
			ontap:    []exchange{cluster, createWithQoS(`{"max_throughput_iops":300,"min_throughput_iops":10,"max_throughput_mbps":50,"min_throughput_mbps":5}`), jobDone},
			wantText: "Volume created successfully",
		},
		{
			name:     "update inline max iops",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos3", map[string]any{"max_iops": "150"}),
			ontap:    []exchange{lookup("docs_qos3", "vol-uuid"), patchQoS(`{"max_throughput_iops":150}`), jobDone},
			wantText: "Volume updated successfully",
		},
		{
			name:     "update inline mbps and min iops",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos", map[string]any{"min_iops": "10", "max_mbps": "50"}),
			ontap:    []exchange{lookup("docs_qos", "vol-uuid"), patchQoS(`{"min_throughput_iops":10,"max_throughput_mbps":50}`), jobDone},
			wantText: "Volume updated successfully",
		},
		{
			// "0" must reach ONTAP as an explicit 0, which removes the limit; omitting it would be a no-op.
			name:     "update max iops 0 removes the inline limit",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos", map[string]any{"max_iops": "0"}),
			ontap:    []exchange{lookup("docs_qos", "vol-uuid"), patchQoS(`{"max_throughput_iops":0}`), jobDone},
			wantText: "Volume updated successfully",
		},
		{
			name:     "remove named policy sends none",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos3", map[string]any{"remove_qos_policy": true}),
			ontap:    []exchange{lookup("docs_qos3", "vol-uuid"), patchQoS(`{"name":"none"}`), jobDone},
			wantText: "Volume updated successfully",
		},
		{
			name:     "apply policy to missing volume",
			tool:     "modify_volume",
			args:     updateArgs("ghost", map[string]any{"policy_name": "qos_docs_200iops"}),
			ontap:    []exchange{lookup("ghost", "")},
			wantText: "failed to update volume=ghost on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "apply missing policy surfaces ONTAP error",
			tool: "modify_volume",
			args: updateArgs("docs_qos", map[string]any{"policy_name": "nope"}),
			ontap: []exchange{
				lookup("docs_qos", "vol-uuid"),
				{method: http.MethodPatch, path: volumesPath + "/vol-uuid", body: `{"qos":{"policy":{"name":"nope"}}}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"QoS policy group \"nope\" does not exist.","code":"917927"}}`},
			},
			wantText: `QoS policy group "nope" does not exist.`,
			wantErr:  true,
		},
		{
			name: "apply policy surfaces async job failure",
			tool: "modify_volume",
			args: updateArgs("docs_qos", map[string]any{"policy_name": "qos_docs_200iops"}),
			ontap: []exchange{
				lookup("docs_qos", "vol-uuid"),
				patchQoS(`{"name":"qos_docs_200iops"}`),
				{method: http.MethodGet, path: "/api/cluster/jobs/qos-vol-job",
					resp: `{"uuid":"qos-vol-job","state":"failure","error":{"message":"Cannot apply a QoS policy group to a volume that has files with QoS policies","code":"13107442"}}`},
			},
			wantText: "Cannot apply a QoS policy group to a volume that has files with QoS policies",
			wantErr:  true,
		},
		{
			name:     "update with negative limit is rejected without calling ONTAP",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos", map[string]any{"max_iops": "-1"}),
			wantText: `invalid max_iops: must be a non-negative integer, got "-1"`,
			wantErr:  true,
		},
		{
			name:     "update with non-numeric limit is rejected without calling ONTAP",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos", map[string]any{"max_mbps": "fast"}),
			wantText: `invalid max_mbps: must be a non-negative integer, got "fast"`,
			wantErr:  true,
		},
		{
			name:     "create with bad limit is rejected before POST",
			tool:     "create_volume",
			args:     createArgs(map[string]any{"min_iops": "10k"}),
			ontap:    []exchange{cluster},
			wantText: `invalid min_iops: must be a non-negative integer, got "10k"`,
			wantErr:  true,
		},
		{
			name:     "update with policy name and inline limit is rejected without calling ONTAP",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos", map[string]any{"policy_name": "p", "max_iops": "100"}),
			wantText: "are mutually exclusive; got policy_name, max_iops",
			wantErr:  true,
		},
		{
			name:     "update with policy name and remove policy is rejected without calling ONTAP",
			tool:     "modify_volume",
			args:     updateArgs("docs_qos", map[string]any{"policy_name": "p", "remove_qos_policy": true}),
			wantText: "are mutually exclusive; got remove_qos_policy, policy_name",
			wantErr:  true,
		},
		{
			name:     "create with policy name and inline limit is rejected before POST",
			tool:     "create_volume",
			args:     createArgs(map[string]any{"policy_name": "p", "max_iops": "100"}),
			ontap:    []exchange{cluster},
			wantText: "are mutually exclusive; got policy_name, max_iops",
			wantErr:  true,
		},
		{
			name:     "create with policy name and remove policy is rejected before POST",
			tool:     "create_volume",
			args:     createArgs(map[string]any{"policy_name": "p", "remove_qos_policy": true}),
			ontap:    []exchange{cluster},
			wantText: "are mutually exclusive; got remove_qos_policy, policy_name",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("got: %q; want to contain: %q", text, tt.wantText)
			}
		})
	}
}
