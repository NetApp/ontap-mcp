package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// fcpInterfaceLookup is the name-to-UUID lookup that precedes every FC interface update and
// delete; an empty uuid means the interface does not exist.
func fcpInterfaceLookup(uuid string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/network/fc/interfaces?svm.name=marketing&name=fc1", resp: `{"num_records":0,"records":[]}`}
	if uuid != "" {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

func TestFCPContract(t *testing.T) {
	const createBody = `{"svm":{"name":"marketing"},"name":"fc1","data_protocol":"fcp","location":{"home_port":{"name":"0e","node":{"name":"umeng-aff300-01"}}}}`
	createArgs := func() map[string]any {
		return map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "fc1", "data_protocol": "fcp",
			"location.home_port.node.name": "umeng-aff300-01", "location.home_port.name": "0e"}
	}
	modifyArgs := func(op string, update map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "operation": op, "svm_name": "marketing", "name": "fc1"}
		if update != nil {
			args["fc_interface_update"] = update
		}
		return args
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
			name: "create FC interface on node port",
			tool: "create_fc_interface",
			args: createArgs(),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/network/fc/interfaces", status: http.StatusCreated, body: createBody},
			},
			wantText: "FC interface created successfully",
		},
		{
			name: "create FC interface surfaces ONTAP error",
			tool: "create_fc_interface",
			args: createArgs(),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/network/fc/interfaces", status: http.StatusBadRequest, body: createBody,
					resp: `{"error":{"message":"Port \"0e\" does not exist on node \"umeng-aff300-01\".","code":"5374810"}}`},
			},
			wantText: `Port "0e" does not exist on node "umeng-aff300-01".`,
			wantErr:  true,
		},
		{
			name:     "create FC interface without home port is rejected without calling ONTAP",
			tool:     "create_fc_interface",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "fc1", "data_protocol": "fcp", "location.home_port.node.name": "umeng-aff300-01", "location.home_port.name": ""},
			wantText: "home port name is required",
			wantErr:  true,
		},
		{
			name:     "create FC interface without data protocol is rejected without calling ONTAP",
			tool:     "create_fc_interface",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "fc1", "data_protocol": "", "location.home_port.node.name": "umeng-aff300-01", "location.home_port.name": "0e"},
			wantText: "data protocol is required",
			wantErr:  true,
		},
		{
			// enabled is sent as the string the caller passed, not a JSON boolean.
			name: "disable FC interface",
			tool: "modify_fc_interface",
			args: modifyArgs("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				fcpInterfaceLookup("fc-uuid"),
				{method: http.MethodPatch, path: "/api/network/fc/interfaces/fc-uuid", body: `{"enabled":"false"}`},
			},
			wantText: "FC interface updated successfully",
		},
		{
			name: "move FC interface to another home port",
			tool: "modify_fc_interface",
			args: modifyArgs("update", map[string]any{"location.home_port.node.name": "umeng-aff300-02", "location.home_port.name": "0f"}),
			ontap: []exchange{
				fcpInterfaceLookup("fc-uuid"),
				{method: http.MethodPatch, path: "/api/network/fc/interfaces/fc-uuid", body: `{"location":{"home_port":{"name":"0f","node":{"name":"umeng-aff300-02"}}}}`},
			},
			wantText: "FC interface updated successfully",
		},
		{
			name: "update of missing FC interface",
			tool: "modify_fc_interface",
			args: modifyArgs("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				fcpInterfaceLookup(""),
			},
			wantText: "failed to get detail of fc interface fc1 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "update FC interface surfaces ONTAP error",
			tool: "modify_fc_interface",
			args: modifyArgs("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				fcpInterfaceLookup("fc-uuid"),
				{method: http.MethodPatch, path: "/api/network/fc/interfaces/fc-uuid", body: `{"enabled":"false"}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"The FC interface cannot be modified while it is enabled.","code":"5374812"}}`},
			},
			wantText: "The FC interface cannot be modified while it is enabled.",
			wantErr:  true,
		},
		{
			name:     "update FC interface with no fields is rejected without calling ONTAP",
			tool:     "modify_fc_interface",
			args:     modifyArgs("update", nil),
			wantText: "at least one supported update field must be provided",
			wantErr:  true,
		},
		{
			name:     "update FC interface with home node but no port is rejected without calling ONTAP",
			tool:     "modify_fc_interface",
			args:     modifyArgs("update", map[string]any{"location.home_port.node.name": "umeng-aff300-02"}),
			wantText: "both location.home_port.node.name and location.home_port.name must be provided together or both omitted",
			wantErr:  true,
		},
		{
			name: "delete FC interface by UUID",
			tool: "modify_fc_interface",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				fcpInterfaceLookup("fc-uuid"),
				{method: http.MethodDelete, path: "/api/network/fc/interfaces/fc-uuid"},
			},
			wantText: "FC interface deleted successfully",
		},
		{
			// The integration suite's "Clean FC Interface" step depends on this message.
			name: "delete of missing FC interface",
			tool: "modify_fc_interface",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				fcpInterfaceLookup(""),
			},
			wantText: "failed to get detail of fc interface fc1 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete with ambiguous lookup does not delete",
			tool: "modify_fc_interface",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/network/fc/interfaces?svm.name=marketing&name=fc1", resp: `{"num_records":2,"records":[{"uuid":"a"},{"uuid":"b"}]}`},
			},
			wantText: "because there are 2 matching records",
			wantErr:  true,
		},
		{
			name: "delete FC interface surfaces ONTAP error",
			tool: "modify_fc_interface",
			args: modifyArgs("delete", nil),
			ontap: []exchange{
				fcpInterfaceLookup("fc-uuid"),
				{method: http.MethodDelete, path: "/api/network/fc/interfaces/fc-uuid", status: http.StatusBadRequest,
					resp: `{"error":{"message":"The FC interface must be disabled before it can be deleted.","code":"5374813"}}`},
			},
			wantText: "The FC interface must be disabled before it can be deleted.",
			wantErr:  true,
		},
		{
			name:     "modify FC interface without name is rejected without calling ONTAP",
			tool:     "modify_fc_interface",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "name": ""},
			wantText: "FC interface name is required",
			wantErr:  true,
		},
		{
			name:     "modify FC interface rejects unknown operation",
			tool:     "modify_fc_interface",
			args:     modifyArgs("disable", nil),
			wantText: `unsupported operation "disable"`,
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
