package server

import (
	"net/http"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

const (
	cdotClusterResp = `{"name":"contract-cluster","uuid":"c-uuid","version":{"full":"NetApp Release 9.16.1","generation":9,"major":16,"minor":1},"san_optimized":false,"disaggregated":false}`
	afxClusterResp  = `{"name":"contract-cluster","uuid":"c-uuid","version":{"full":"NetApp Release 9.17.1","generation":9,"major":17,"minor":1},"san_optimized":false,"disaggregated":true}`
	jobAccepted     = `{"job":{"uuid":"job-1"}}`
)

func getCluster(resp string) exchange {
	return exchange{method: http.MethodGet, path: "/api/cluster?fields=*", resp: resp}
}

// lookupVolume is the name-to-UUID lookup that precedes every update and delete; an empty uuid
// means the volume does not exist.
func lookupVolume(volume, uuid string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/storage/volumes?fields=uuid&svm.name=marketing&name=" + volume}
	if uuid == "" {
		e.resp = `{"num_records":0,"records":[]}`
	} else {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

var jobSuccess = exchange{method: http.MethodGet, path: "/api/cluster/jobs/job-1", resp: `{"uuid":"job-1","state":"success"}`}

func TestVolumeContract(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			name: "create flexvol on CDOT",
			tool: "create_volume",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "aggregate_name": "aggr1", "size": "20MB", "nas.path": "/docs"},
			ontap: []exchange{
				getCluster(cdotClusterResp),
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusAccepted, resp: jobAccepted,
					body: `{"svm":{"name":"marketing"},"name":"docs","aggregates":[{"name":"aggr1"}],"size":20971520,"nas":{"path":"/docs"}}`},
				jobSuccess,
			},
			wantText: "Volume created successfully",
		},
		{
			name: "create volume with tiering policy none",
			tool: "create_volume",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "cold", "aggregate_name": "aggr1", "tiering.policy": "none"},
			ontap: []exchange{
				getCluster(cdotClusterResp),
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusAccepted, resp: jobAccepted,
					body: `{"svm":{"name":"marketing"},"name":"cold","aggregates":[{"name":"aggr1"}],"tiering":{"policy":"none"}}`},
				jobSuccess,
			},
			wantText: "Volume created successfully",
		},
		{
			name: "create on AFX omits aggregates",
			tool: "create_volume",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "size": "50MB", "space.snapshot.reserve_percent": 5},
			ontap: []exchange{
				getCluster(afxClusterResp),
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusAccepted, resp: jobAccepted,
					body: `{"svm":{"name":"marketing"},"name":"docs","size":52428800,"space":{"snapshot":{"reserve_percent":5}}}`},
				jobSuccess,
			},
			wantText: "Volume created successfully",
		},
		{
			name: "create thick volume with policy and efficiency settings",
			tool: "create_volume",
			args: map[string]any{
				"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "thick", "aggregate_name": "aggr1", "size": "50MB",
				"guarantee.type": "volume", "snapshot_policy.name": "none", "efficiency": map[string]any{"dedupe": "none", "compression": "none"},
			},
			ontap: []exchange{
				getCluster(cdotClusterResp),
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusAccepted, resp: jobAccepted,
					body: `{"svm":{"name":"marketing"},"name":"thick","aggregates":[{"name":"aggr1"}],"size":52428800,
						"guarantee":{"type":"volume"},"snapshot_policy":{"name":"none"},"efficiency":{"dedupe":"none","compression":"none"}}`},
				jobSuccess,
			},
			wantText: "Volume created successfully",
		},
		{
			name: "create with missing export policy creates it first",
			tool: "create_volume",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "aggregate_name": "aggr1", "nas.export_policy.name": "pol1"},
			ontap: []exchange{
				getCluster(cdotClusterResp),
				{method: http.MethodGet, path: "/api/protocols/nfs/export-policies?fields=name&name=pol1&return_records=false&svm.name=marketing", resp: `{"num_records":0}`},
				{method: http.MethodPost, path: "/api/protocols/nfs/export-policies", status: http.StatusCreated, body: `{"name":"pol1","svm":{"name":"marketing"}}`},
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusAccepted, resp: jobAccepted,
					body: `{"svm":{"name":"marketing"},"name":"docs","aggregates":[{"name":"aggr1"}],"nas":{"export_policy":{"name":"pol1"}}}`},
				jobSuccess,
			},
			wantText: "Volume created successfully",
		},
		{
			name:     "create on CDOT without aggregate is rejected before POST",
			tool:     "create_volume",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs"},
			ontap:    []exchange{getCluster(cdotClusterResp)},
			wantText: "aggregate name is required",
			wantErr:  true,
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_volume",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "aggregate_name": "aggr1"},
			ontap: []exchange{
				getCluster(cdotClusterResp),
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusConflict,
					body: `{"svm":{"name":"marketing"},"name":"docs","aggregates":[{"name":"aggr1"}]}`,
					resp: `{"error":{"message":"Duplicate volume name docs.","code":"917835"}}`},
			},
			wantText: "Duplicate volume name docs.",
			wantErr:  true,
		},
		{
			name: "create surfaces async job failure",
			tool: "create_volume",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "aggregate_name": "aggr1"},
			ontap: []exchange{
				getCluster(cdotClusterResp),
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusAccepted, resp: jobAccepted,
					body: `{"svm":{"name":"marketing"},"name":"docs","aggregates":[{"name":"aggr1"}]}`},
				{method: http.MethodGet, path: "/api/cluster/jobs/job-1",
					resp: `{"uuid":"job-1","state":"failure","error":{"message":"Insufficient space in aggregate aggr1","code":"917927"}}`},
			},
			wantText: "Insufficient space in aggregate aggr1",
			wantErr:  true,
		},
		{
			name: "modify resizes by UUID",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs", "volume_update": map[string]any{"size": "25MB"}},
			ontap: []exchange{
				lookupVolume("docs", "vol-uuid"),
				{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"size":26214400}`, status: http.StatusAccepted, resp: jobAccepted},
				jobSuccess,
			},
			wantText: "Volume updated successfully",
		},
		{
			name: "modify renames",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs", "volume_update": map[string]any{"new_volume_name": "docsnew"}},
			ontap: []exchange{
				lookupVolume("docs", "vol-uuid"),
				{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"name":"docsnew"}`, status: http.StatusAccepted, resp: jobAccepted},
				jobSuccess,
			},
			wantText: "Volume updated successfully",
		},
		{
			name: "modify takes volume offline",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs", "volume_update": map[string]any{"state": "offline"}},
			ontap: []exchange{
				lookupVolume("docs", "vol-uuid"),
				{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"state":"offline"}`},
			},
			wantText: "Volume updated successfully",
		},
		{
			name: "modify enables autogrow",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs",
				"volume_update": map[string]any{"autosize": map[string]any{"mode": "grow", "grow_threshold": "62"}}},
			ontap: []exchange{
				lookupVolume("docs", "vol-uuid"),
				{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"autosize":{"mode":"grow","grow_threshold":"62"}}`},
			},
			wantText: "Volume updated successfully",
		},
		{
			name: "modify autosize threshold without mode omits mode",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs",
				"volume_update": map[string]any{"autosize": map[string]any{"grow_threshold": "90"}}},
			ontap: []exchange{
				lookupVolume("docs", "vol-uuid"),
				{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"autosize":{"grow_threshold":"90"}}`},
			},
			wantText: "Volume updated successfully",
		},
		{
			name: "modify sets files maximum",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs", "volume_update": map[string]any{"files.maximum": 2000}},
			ontap: []exchange{
				lookupVolume("docs", "vol-uuid"),
				{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"files":{"maximum":2000}}`},
			},
			wantText: "Volume updated successfully",
		},
		{
			name:     "modify with no fields is rejected without calling ONTAP",
			tool:     "modify_volume",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs"},
			wantText: "at least one updatable field must be provided",
			wantErr:  true,
		},
		{
			name: "modify update of missing volume",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "ghost", "volume_update": map[string]any{"size": "25MB"}},
			ontap: []exchange{
				lookupVolume("ghost", ""),
			},
			wantText: "failed to update volume=ghost on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify deletes by UUID",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "volume_name": "docs"},
			ontap: []exchange{
				lookupVolume("docs", "vol-uuid"),
				{method: http.MethodDelete, path: "/api/storage/volumes/vol-uuid", status: http.StatusAccepted, resp: jobAccepted},
				jobSuccess,
			},
			wantText: "Volume deleted successfully",
		},
		{
			// The integration suite's "Clean volume" steps depend on this exact message.
			name: "modify delete of missing volume",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "volume_name": "ghost"},
			ontap: []exchange{
				lookupVolume("ghost", ""),
			},
			wantText: "because it does not exist",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_volume",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "resize", "svm_name": "marketing", "volume_name": "docs"},
			wantText: `unsupported operation "resize"`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_volume",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "marketing", "volume_name": "docs"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			assert.Contains(t, text, tt.wantText)
		})
	}
}

func TestUpdateVolumeTieringPolicy(t *testing.T) {
	cs := newContractSession(t, "both", []exchange{
		lookupVolume("docs", "vol-uuid"),
		{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"tiering":{"policy":"auto"}}`},
	})
	text, isErr := callTool(t, cs, "update_volume", map[string]any{
		"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "tiering.policy": "auto",
	})
	if isErr {
		t.Fatalf("update_volume error = %s", text)
	}
	if text != "Volume updated successfully" {
		t.Errorf("update_volume text = %q, want success", text)
	}
}
