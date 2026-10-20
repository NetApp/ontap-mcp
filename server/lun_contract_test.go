package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

var lunClusterInfo = exchange{
	method: http.MethodGet,
	path:   "/api/cluster?fields=*",
	resp:   `{"name":"` + contractCluster + `"}`,
}

// lunLookup is the name-to-UUID lookup that precedes every LUN update and delete; an empty uuid
// means the LUN does not exist.
func lunLookup(path, uuid string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/storage/luns?fields=uuid&svm.name=marketing&name=" + path,
		resp: `{"num_records":0,"records":[]}`}
	if uuid != "" {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

func TestLUNContract(t *testing.T) {
	const (
		lunPath     = "/vol/doc/lundoc"
		lunJob      = `{"job":{"uuid":"lun-job-1"}}`
		lunJobPath  = "/api/cluster/jobs/lun-job-1"
		createdBody = `{"svm":{"name":"marketing"},"name":"/vol/doc/lundoc","space":{"size":20971520},"os_type":"linux"}`
	)
	createArgs := func() map[string]any {
		return map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc", "size": "20MB", "os_type": "linux"}
	}
	modifyArgs := func(op string, update map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "operation": op, "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc"}
		if update != nil {
			args["lun_update"] = update
		}
		return args
	}

	tests := []struct {
		name     string
		mode     string // tool mode; empty means "multiplex"
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			name: "create LUN with OS type",
			tool: "create_lun",
			args: createArgs(),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/luns", status: http.StatusCreated, body: createdBody},
			},
			wantText: "LUN created successfully",
		},
		{
			name: "create LUN waits for async job",
			tool: "create_lun",
			args: createArgs(),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/luns", status: http.StatusAccepted, resp: lunJob, body: createdBody},
				{method: http.MethodGet, path: lunJobPath, resp: `{"uuid":"lun-job-1","state":"success"}`},
			},
			wantText: "LUN created successfully",
		},
		{
			name: "create thick-provisioned LUN requests space guarantee",
			tool: "create_lun",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundocthick",
				"size": "10MB", "os_type": "linux", "space_guarantee_requested": true},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/luns", status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"/vol/doc/lundocthick","space":{"size":10485760,"guarantee":{"requested":true}},"os_type":"linux"}`},
			},
			wantText: "LUN created successfully",
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_lun",
			args: createArgs(),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/luns", status: http.StatusBadRequest, body: createdBody,
					resp: `{"error":{"message":"Volume \"doc\" in SVM \"marketing\" does not exist.","code":"5374863"}}`},
			},
			wantText: `Volume "doc" in SVM "marketing" does not exist.`,
			wantErr:  true,
		},
		{
			name: "create surfaces async job failure",
			tool: "create_lun",
			args: createArgs(),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/luns", status: http.StatusAccepted, resp: lunJob, body: createdBody},
				{method: http.MethodGet, path: lunJobPath,
					resp: `{"uuid":"lun-job-1","state":"failure","error":{"message":"Insufficient space in volume doc","code":"5374877"}}`},
			},
			wantText: "job failed code=5374877 msg=Insufficient space in volume doc",
			wantErr:  true,
		},
		{
			name:     "create without OS type is rejected without calling ONTAP",
			tool:     "create_lun",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc", "size": "20MB", "os_type": ""},
			wantText: "OS type is required",
			wantErr:  true,
		},
		{
			name:     "create with bad size is rejected without calling ONTAP",
			tool:     "create_lun",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc", "size": "20XB", "os_type": "linux"},
			wantText: "invalid size",
			wantErr:  true,
		},
		{
			name: "modify resizes LUN by UUID",
			tool: "modify_lun",
			args: modifyArgs("update", map[string]any{"size": "50mb"}),
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodPatch, path: "/api/storage/luns/lun-uuid", body: `{"space":{"size":52428800}}`},
			},
			wantText: "LUN updated successfully",
		},
		{
			name: "modify renames LUN within its volume",
			tool: "modify_lun",
			args: modifyArgs("update", map[string]any{"new_lun_name": "lundocnew"}),
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodPatch, path: "/api/storage/luns/lun-uuid", body: `{"name":"/vol/doc/lundocnew"}`},
			},
			wantText: "LUN updated successfully",
		},
		{
			name: "modify disables LUN",
			tool: "modify_lun",
			args: modifyArgs("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodPatch, path: "/api/storage/luns/lun-uuid", body: `{"enabled":false}`},
			},
			wantText: "LUN updated successfully",
		},
		{
			name: "modify update of missing LUN",
			tool: "modify_lun",
			args: modifyArgs("update", map[string]any{"size": "50MB"}),
			ontap: []exchange{
				lunLookup(lunPath, ""),
			},
			wantText: "failed to get lun=/vol/doc/lundoc on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify update surfaces ONTAP error",
			tool: "modify_lun",
			args: modifyArgs("update", map[string]any{"size": "1MB"}),
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodPatch, path: "/api/storage/luns/lun-uuid", body: `{"space":{"size":1048576}}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Reducing the size of a LUN might cause permanent data loss.","code":"5374922"}}`},
			},
			wantText: "Reducing the size of a LUN might cause permanent data loss.",
			wantErr:  true,
		},
		{
			name:     "modify update with no fields is rejected without calling ONTAP",
			tool:     "modify_lun",
			args:     modifyArgs("update", nil),
			wantText: "at least one of new_lun_name, size, or enabled must be provided",
			wantErr:  true,
		},
		{
			name:     "modify update with bad enabled value is rejected without calling ONTAP",
			tool:     "modify_lun",
			args:     modifyArgs("update", map[string]any{"enabled": "off"}),
			wantText: `invalid enabled value "off"`,
			wantErr:  true,
		},
		{
			name: "modify deletes LUN by UUID",
			tool: "modify_lun",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodDelete, path: "/api/storage/luns/lun-uuid?allow_delete_while_mapped=false"},
			},
			wantText: "LUN deleted successfully",
		},
		{
			name: "modify deletes mapped LUN when allowed",
			tool: "modify_lun",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc", "allow_delete_while_mapped": true},
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodDelete, path: "/api/storage/luns/lun-uuid?allow_delete_while_mapped=true"},
			},
			wantText: "LUN deleted successfully",
		},
		{
			// The integration suite's "Clean LUN" steps depend on this message.
			name: "modify delete of missing LUN",
			tool: "modify_lun",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				lunLookup(lunPath, ""),
			},
			wantText: "failed to get lun=/vol/doc/lundoc on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify delete with ambiguous lookup does not delete",
			tool: "modify_lun",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/storage/luns?fields=uuid&svm.name=marketing&name=" + lunPath,
					resp: `{"num_records":2,"records":[{"uuid":"a"},{"uuid":"b"}]}`},
			},
			wantText: "because there are 2 matching records",
			wantErr:  true,
		},
		{
			name: "modify delete surfaces ONTAP error from lookup",
			tool: "modify_lun",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/storage/luns?fields=uuid&svm.name=marketing&name=" + lunPath, status: http.StatusNotFound,
					resp: `{"error":{"message":"SVM \"marketing\" does not exist.","code":"2621462"}}`},
			},
			wantText: `SVM "marketing" does not exist.`,
			wantErr:  true,
		},
		{
			name: "modify delete surfaces ONTAP error from delete",
			tool: "modify_lun",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodDelete, path: "/api/storage/luns/lun-uuid?allow_delete_while_mapped=false", status: http.StatusBadRequest,
					resp: `{"error":{"message":"The LUN is mapped and cannot be deleted.","code":"5374878"}}`},
			},
			wantText: "The LUN is mapped and cannot be deleted.",
			wantErr:  true,
		},
		{
			name: "modify deletes thick-provisioned LUN",
			tool: "modify_lun",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundocthick"},
			ontap: []exchange{
				lunLookup("/vol/doc/lundocthick", "thick-uuid"),
				{method: http.MethodDelete, path: "/api/storage/luns/thick-uuid?allow_delete_while_mapped=false"},
			},
			wantText: "LUN deleted successfully",
		},
		{
			name: "legacy update_lun renames and resizes in one PATCH",
			mode: "legacy",
			tool: "update_lun",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc", "new_lun_name": "lundocnew", "size": "50MB"},
			ontap: []exchange{
				lunLookup(lunPath, "lun-uuid"),
				{method: http.MethodPatch, path: "/api/storage/luns/lun-uuid", body: `{"name":"/vol/doc/lundocnew","space":{"size":52428800}}`},
			},
			wantText: "LUN updated successfully",
		},
		{
			name: "legacy delete_lun of missing LUN",
			mode: "legacy",
			tool: "delete_lun",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc"},
			ontap: []exchange{
				lunLookup(lunPath, ""),
			},
			wantText: "failed to get lun=/vol/doc/lundoc on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_lun",
			args:     modifyArgs("resize", nil),
			wantText: `unsupported operation "resize"`,
			wantErr:  true,
		},
		{
			name:     "modify without volume is rejected without calling ONTAP",
			tool:     "modify_lun",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "volume_name": "", "lun_name": "lundoc"},
			wantText: "volume name is required",
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_lun",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "marketing", "volume_name": "doc", "lun_name": "lundoc"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := tt.mode
			if mode == "" {
				mode = "multiplex"
			}
			script := tt.ontap
			if cluster, ok := tt.args["cluster_name"].(string); ok && cluster == contractCluster {
				script = append([]exchange{lunClusterInfo}, script...)
			}
			cs := newContractSession(t, mode, script)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("result %q does not contain %q", text, tt.wantText)
			}
		})
	}
}
