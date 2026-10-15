package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// igroupLookup is the name-to-UUID lookup that precedes every igroup update, delete, and initiator
// change; an empty uuid means the igroup does not exist.
func igroupLookup(name, uuid string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/protocols/san/igroups?fields=uuid&svm.name=marketing&name=" + name,
		resp: `{"num_records":0,"records":[]}`}
	if uuid != "" {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

func TestIGroupLUNMapContract(t *testing.T) {
	const (
		iqn          = "iqn.2021-01.com.example:test"
		lunMapLookup = "/api/protocols/san/lun-maps?fields=lun.uuid,igroup.uuid&svm.name=marketing&lun.name=/vol/doc/lundoc&igroup.name=igroupFinNew"
		lunMapBody   = `{"svm":{"name":"marketing"},"lun":{"name":"/vol/doc/lundoc"},"igroup":{"name":"igroupFinNew"}}`
	)
	initiatorArgs := map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "igroup_name": "igroupFin", "initiator_name": iqn}
	lunMapArgs := map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "lun_name": "/vol/doc/lundoc", "igroup_name": "igroupFinNew"}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		// igroup create
		{
			name: "create igroup with initiators atomically",
			tool: "create_igroup",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "igroupFin", "os_type": "linux", "protocol": "iscsi", "initiators": []any{iqn}},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/igroups", status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"igroupFin","os_type":"linux","protocol":"iscsi","initiators":[{"name":"` + iqn + `"}]}`},
			},
			wantText: "igroup created successfully",
		},
		{
			name: "create igroup normalizes protocol and trims initiators",
			tool: "create_igroup",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "igroupFin", "os_type": "vmware", "protocol": " FCP ",
				"comment": "esx hosts", "initiators": []any{" 20:00:00:25:b5:00:00:01 ", "20:00:00:25:b5:00:00:02"}},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/igroups", status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"igroupFin","os_type":"vmware","protocol":"fcp","comment":"esx hosts",
						"initiators":[{"name":"20:00:00:25:b5:00:00:01"},{"name":"20:00:00:25:b5:00:00:02"}]}`},
			},
			wantText: "igroup created successfully",
		},
		{
			name: "create igroup surfaces async job failure",
			tool: "create_igroup",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "igroupFin", "os_type": "linux", "protocol": "iscsi"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/igroups", status: http.StatusAccepted, resp: `{"job":{"uuid":"igroup-job-1"}}`,
					body: `{"svm":{"name":"marketing"},"name":"igroupFin","os_type":"linux","protocol":"iscsi"}`},
				{method: http.MethodGet, path: "/api/cluster/jobs/igroup-job-1",
					resp: `{"uuid":"igroup-job-1","state":"failure","error":{"message":"igroup create failed","code":"5374733"}}`},
			},
			wantText: "job failed code=5374733 msg=igroup create failed",
			wantErr:  true,
		},
		{
			name: "create igroup surfaces ONTAP error message",
			tool: "create_igroup",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "igroupFin", "os_type": "linux", "protocol": "iscsi"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/igroups", status: http.StatusConflict,
					body: `{"svm":{"name":"marketing"},"name":"igroupFin","os_type":"linux","protocol":"iscsi"}`,
					resp: `{"error":{"message":"The initiator group already exists.","code":"5374734"}}`},
			},
			wantText: "The initiator group already exists.",
			wantErr:  true,
		},
		{
			name:     "create igroup with unsupported protocol is rejected without calling ONTAP",
			tool:     "create_igroup",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "igroupFin", "os_type": "linux", "protocol": "nvme"},
			wantText: `unsupported igroup protocol "nvme"`,
			wantErr:  true,
		},
		{
			name:     "create igroup with blank initiator is rejected without calling ONTAP",
			tool:     "create_igroup",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "igroupFin", "os_type": "linux", "protocol": "iscsi", "initiators": []any{"  "}},
			wantText: "all initiators must be non-empty and non-whitespace",
			wantErr:  true,
		},

		// initiators
		{
			name: "remove initiator from igroup",
			tool: "remove_igroup_initiator",
			args: initiatorArgs,
			ontap: []exchange{
				igroupLookup("igroupFin", "ig-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/igroups/ig-uuid/initiators/" + iqn + "?allow_delete_while_mapped=false"},
			},
			wantText: "initiator removed from igroup successfully",
		},
		{
			name: "remove initiator from mapped igroup when allowed",
			tool: "remove_igroup_initiator",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "igroup_name": "igroupFin", "initiator_name": iqn, "allow_delete_while_mapped": true},
			ontap: []exchange{
				igroupLookup("igroupFin", "ig-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/igroups/ig-uuid/initiators/" + iqn + "?allow_delete_while_mapped=true"},
			},
			wantText: "initiator removed from igroup successfully",
		},
		{
			name: "remove initiator surfaces ONTAP error",
			tool: "remove_igroup_initiator",
			args: initiatorArgs,
			ontap: []exchange{
				igroupLookup("igroupFin", "ig-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/igroups/ig-uuid/initiators/" + iqn + "?allow_delete_while_mapped=false", status: http.StatusNotFound,
					resp: `{"error":{"message":"The initiator is not a member of the initiator group.","code":"5374852"}}`},
			},
			wantText: "The initiator is not a member of the initiator group.",
			wantErr:  true,
		},
		{
			name:     "remove initiator without initiator name is rejected without calling ONTAP",
			tool:     "remove_igroup_initiator",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "igroup_name": "igroupFin"},
			wantText: "initiator name is required",
			wantErr:  true,
		},
		{
			name: "add initiator to igroup",
			tool: "add_igroup_initiator",
			args: initiatorArgs,
			ontap: []exchange{
				igroupLookup("igroupFin", "ig-uuid"),
				{method: http.MethodPost, path: "/api/protocols/san/igroups/ig-uuid/initiators", status: http.StatusCreated, body: `{"name":"` + iqn + `"}`},
			},
			wantText: "initiator added to igroup successfully",
		},
		{
			name: "add several initiators in one call",
			tool: "add_igroup_initiator",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "igroup_name": "igroupFin", "records": []any{iqn, "iqn.2021-01.com.example:other"}},
			ontap: []exchange{
				igroupLookup("igroupFin", "ig-uuid"),
				{method: http.MethodPost, path: "/api/protocols/san/igroups/ig-uuid/initiators", status: http.StatusCreated,
					body: `{"records":[{"name":"` + iqn + `"},{"name":"iqn.2021-01.com.example:other"}]}`},
			},
			wantText: "initiator added to igroup successfully",
		},
		{
			name: "add initiator to missing igroup",
			tool: "add_igroup_initiator",
			args: initiatorArgs,
			ontap: []exchange{
				igroupLookup("igroupFin", ""),
			},
			wantText: "failed to get igroup=igroupFin on svm=marketing because the igroup does not exist",
			wantErr:  true,
		},
		{
			name:     "add initiator with both name and records is rejected without calling ONTAP",
			tool:     "add_igroup_initiator",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "igroup_name": "igroupFin", "initiator_name": iqn, "records": []any{iqn}},
			wantText: "specify either a single initiator name or an array of initiator names, but not both",
			wantErr:  true,
		},
		{
			name:     "add initiator with neither name nor records is rejected without calling ONTAP",
			tool:     "add_igroup_initiator",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "igroup_name": "igroupFin"},
			wantText: "either initiator name OR one or more initiator names (records) must be provided",
			wantErr:  true,
		},

		// igroup update and delete
		{
			name: "rename igroup",
			tool: "modify_igroup",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "name": "igroupFin", "igroup_update": map[string]any{"new_name": "igroupFinNew"}},
			ontap: []exchange{
				igroupLookup("igroupFin", "ig-uuid"),
				{method: http.MethodPatch, path: "/api/protocols/san/igroups/ig-uuid", body: `{"name":"igroupFinNew"}`},
			},
			wantText: "igroup updated successfully",
		},
		{
			name: "update igroup OS type and comment",
			tool: "modify_igroup",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "name": "igroupFin",
				"igroup_update": map[string]any{"os_type": "windows", "comment": "moved"}},
			ontap: []exchange{
				igroupLookup("igroupFin", "ig-uuid"),
				{method: http.MethodPatch, path: "/api/protocols/san/igroups/ig-uuid", body: `{"os_type":"windows","comment":"moved"}`},
			},
			wantText: "igroup updated successfully",
		},
		{
			name: "update of missing igroup",
			tool: "modify_igroup",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "name": "igroupFin", "igroup_update": map[string]any{"new_name": "igroupFinNew"}},
			ontap: []exchange{
				igroupLookup("igroupFin", ""),
			},
			wantText: "failed to get igroup=igroupFin on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name:     "update igroup with no fields is rejected without calling ONTAP",
			tool:     "modify_igroup",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "name": "igroupFin"},
			wantText: "at least one updatable field must be provided: new_name, comment, or os_type",
			wantErr:  true,
		},
		{
			name: "delete igroup by UUID",
			tool: "modify_igroup",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "name": "igroupFinNew"},
			ontap: []exchange{
				igroupLookup("igroupFinNew", "ig-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/igroups/ig-uuid?allow_delete_while_mapped=false"},
			},
			wantText: "igroup deleted successfully",
		},
		{
			name: "delete mapped igroup when allowed",
			tool: "modify_igroup",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "name": "igroupFinNew", "allow_delete_while_mapped": true},
			ontap: []exchange{
				igroupLookup("igroupFinNew", "ig-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/igroups/ig-uuid?allow_delete_while_mapped=true"},
			},
			wantText: "igroup deleted successfully",
		},
		{
			// The integration suite's "Clean igroup" steps depend on this message.
			name: "delete of missing igroup",
			tool: "modify_igroup",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "name": "igroupFin"},
			ontap: []exchange{
				igroupLookup("igroupFin", ""),
			},
			wantText: "failed to get igroup=igroupFin on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete surfaces ONTAP error",
			tool: "modify_igroup",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "name": "igroupFinNew"},
			ontap: []exchange{
				igroupLookup("igroupFinNew", "ig-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/igroups/ig-uuid?allow_delete_while_mapped=false", status: http.StatusBadRequest,
					resp: `{"error":{"message":"The initiator group is mapped to one or more LUNs.","code":"5374737"}}`},
			},
			wantText: "The initiator group is mapped to one or more LUNs.",
			wantErr:  true,
		},
		{
			name:     "modify igroup rejects unknown operation",
			tool:     "modify_igroup",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "rename", "svm_name": "marketing", "name": "igroupFin"},
			wantText: `unsupported operation "rename"`,
			wantErr:  true,
		},

		// LUN maps
		{
			name: "create lun map",
			tool: "create_lun_map",
			args: lunMapArgs,
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/lun-maps", status: http.StatusCreated, body: lunMapBody,
					resp: `{"num_records":1,"records":[{"logical_unit_number":0}]}`},
			},
			wantText: "lun map created successfully",
		},
		{
			name: "create lun map surfaces ONTAP error",
			tool: "create_lun_map",
			args: lunMapArgs,
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/lun-maps", status: http.StatusBadRequest, body: lunMapBody,
					resp: `{"error":{"message":"The specified LUN does not exist or is not accessible to the caller.","code":"5374875"}}`},
			},
			wantText: "The specified LUN does not exist or is not accessible to the caller.",
			wantErr:  true,
		},
		{
			name:     "create lun map without igroup is rejected without calling ONTAP",
			tool:     "create_lun_map",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "lun_name": "/vol/doc/lundoc", "igroup_name": ""},
			wantText: "igroup name is required",
			wantErr:  true,
		},
		{
			name: "delete lun map by LUN and igroup UUIDs",
			tool: "delete_lun_map",
			args: lunMapArgs,
			ontap: []exchange{
				{method: http.MethodGet, path: lunMapLookup,
					resp: `{"num_records":1,"records":[{"lun":{"uuid":"lun-uuid"},"igroup":{"uuid":"ig-uuid"}}]}`},
				{method: http.MethodDelete, path: "/api/protocols/san/lun-maps/lun-uuid/ig-uuid"},
			},
			wantText: "lun map deleted successfully",
		},
		{
			name: "delete of missing lun map",
			tool: "delete_lun_map",
			args: lunMapArgs,
			ontap: []exchange{
				{method: http.MethodGet, path: lunMapLookup, resp: `{"num_records":0,"records":[]}`},
			},
			wantText: "failed to find lun map for lun=/vol/doc/lundoc igroup=igroupFinNew on svm=marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete lun map surfaces ONTAP error",
			tool: "delete_lun_map",
			args: lunMapArgs,
			ontap: []exchange{
				{method: http.MethodGet, path: lunMapLookup,
					resp: `{"num_records":1,"records":[{"lun":{"uuid":"lun-uuid"},"igroup":{"uuid":"ig-uuid"}}]}`},
				{method: http.MethodDelete, path: "/api/protocols/san/lun-maps/lun-uuid/ig-uuid", status: http.StatusForbidden,
					resp: `{"error":{"message":"not authorized for that command","code":"6"}}`},
			},
			wantText: "not authorized for that command",
			wantErr:  true,
		},
		{
			name:     "delete lun map without LUN name is rejected without calling ONTAP",
			tool:     "delete_lun_map",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "lun_name": "", "igroup_name": "igroupFinNew"},
			wantText: "LUN name is required",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("result %q does not contain %q", text, tt.wantText)
			}
		})
	}
}
