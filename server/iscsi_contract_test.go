package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// iscsiServiceLookup finds the iSCSI service by SVM name; the service is addressed by its SVM UUID.
// An empty svmUUID means the service does not exist.
func iscsiServiceLookup(svmUUID string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/protocols/san/iscsi/services?svm.name=marketing", resp: `{"num_records":0,"records":[]}`}
	if svmUUID != "" {
		e.resp = `{"num_records":1,"records":[{"svm":{"name":"marketing","uuid":"` + svmUUID + `"}}]}`
	}
	return e
}

// iscsiLIFLookup is the network interface lookup that precedes every LIF update and delete. svm is
// only sent for svm-scoped interfaces. An empty uuid means the interface does not exist.
func iscsiLIFLookup(name, scope, svm, uuid string) exchange {
	path := "/api/network/ip/interfaces?name=" + name + "&scope=" + scope
	if svm != "" {
		path += "&svm.name=" + svm
	}
	e := exchange{method: http.MethodGet, path: path, resp: `{"num_records":0,"records":[]}`}
	if uuid != "" {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

func TestIscsiContract(t *testing.T) {
	modifyService := func(op string, update map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "operation": op, "svm_name": "marketing"}
		if update != nil {
			args["iscsi_service_update"] = update
		}
		return args
	}
	deleteLIF := func(name, scope string) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "operation": "delete", "name": name, "scope": scope}
		if scope == "svm" {
			args["svm_name"] = "marketing"
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
		// iSCSI service
		{
			name: "create iSCSI service with target alias",
			tool: "create_iscsi_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "target.alias": "tgpath"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/iscsi/services", status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"target":{"alias":"tgpath"}}`},
			},
			wantText: "iSCSI Service created successfully",
		},
		{
			// TestIGroupLUNMap creates the service with no options.
			name: "create iSCSI service with defaults",
			tool: "create_iscsi_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/iscsi/services", status: http.StatusCreated, body: `{"svm":{"name":"marketing"}}`},
			},
			wantText: "iSCSI Service created successfully",
		},
		{
			name: "create iSCSI service surfaces ONTAP error",
			tool: "create_iscsi_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/protocols/san/iscsi/services", status: http.StatusConflict, body: `{"svm":{"name":"marketing"}}`,
					resp: `{"error":{"message":"An iSCSI service already exists for SVM \"marketing\".","code":"5374894"}}`},
			},
			wantText: `An iSCSI service already exists for SVM "marketing".`,
			wantErr:  true,
		},
		{
			name:     "create iSCSI service without SVM is rejected without calling ONTAP",
			tool:     "create_iscsi_service",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			// enabled is sent as the string the caller passed, not a JSON boolean.
			name: "disable iSCSI service admin state",
			tool: "modify_iscsi_service",
			args: modifyService("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				iscsiServiceLookup("svm-uuid"),
				{method: http.MethodPatch, path: "/api/protocols/san/iscsi/services/svm-uuid", body: `{"enabled":"false"}`},
			},
			wantText: "iSCSI Service updated successfully",
		},
		{
			name: "update of missing iSCSI service",
			tool: "modify_iscsi_service",
			args: modifyService("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				iscsiServiceLookup(""),
			},
			wantText: "failed to get detail of iscsi service in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "update iSCSI service surfaces ONTAP error",
			tool: "modify_iscsi_service",
			args: modifyService("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				iscsiServiceLookup("svm-uuid"),
				{method: http.MethodPatch, path: "/api/protocols/san/iscsi/services/svm-uuid", body: `{"enabled":"false"}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Invalid value for enabled","code":"262179"}}`},
			},
			wantText: "Invalid value for enabled",
			wantErr:  true,
		},
		{
			name:     "update iSCSI service without fields is rejected without calling ONTAP",
			tool:     "modify_iscsi_service",
			args:     modifyService("update", nil),
			wantText: "at least one field must be provided for update",
			wantErr:  true,
		},
		{
			name:     "update iSCSI service with empty update object is rejected without calling ONTAP",
			tool:     "modify_iscsi_service",
			args:     modifyService("update", map[string]any{}),
			wantText: "at least one field must be provided for update",
			wantErr:  true,
		},
		{
			name: "update iSCSI service refuses ambiguous match",
			tool: "modify_iscsi_service",
			args: modifyService("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/protocols/san/iscsi/services?svm.name=marketing",
					resp: `{"num_records":2,"records":[{"svm":{"name":"marketing","uuid":"a"}},{"svm":{"name":"marketing","uuid":"b"}}]}`},
			},
			wantText: "failed to get iscsi service on svm=marketing because there are 2 matching records",
			wantErr:  true,
		},
		{
			name: "update iSCSI service with empty records is treated as missing",
			tool: "modify_iscsi_service",
			args: modifyService("update", map[string]any{"enabled": "false"}),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/protocols/san/iscsi/services?svm.name=marketing", resp: `{"num_records":1,"records":[]}`},
			},
			wantText: "failed to get detail of iscsi service in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete iSCSI service by SVM UUID",
			tool: "modify_iscsi_service",
			args: modifyService("delete", nil),
			ontap: []exchange{
				iscsiServiceLookup("svm-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/iscsi/services/svm-uuid"},
			},
			wantText: "iSCSI Service deleted successfully",
		},
		{
			// The integration suite's "Clean iSCSI service" steps depend on this message.
			name: "delete of missing iSCSI service",
			tool: "modify_iscsi_service",
			args: modifyService("delete", nil),
			ontap: []exchange{
				iscsiServiceLookup(""),
			},
			wantText: "failed to get detail of iscsi service in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete iSCSI service refuses ambiguous match",
			tool: "modify_iscsi_service",
			args: modifyService("delete", nil),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/protocols/san/iscsi/services?svm.name=marketing",
					resp: `{"num_records":2,"records":[{"svm":{"name":"marketing","uuid":"a"}},{"svm":{"name":"marketing","uuid":"b"}}]}`},
			},
			wantText: "failed to get iscsi service on svm=marketing because there are 2 matching records",
			wantErr:  true,
		},
		{
			name: "delete iSCSI service with empty records is treated as missing",
			tool: "modify_iscsi_service",
			args: modifyService("delete", nil),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/protocols/san/iscsi/services?svm.name=marketing", resp: `{"num_records":1,"records":[]}`},
			},
			wantText: "failed to get detail of iscsi service in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete iSCSI service surfaces ONTAP error",
			tool: "modify_iscsi_service",
			args: modifyService("delete", nil),
			ontap: []exchange{
				iscsiServiceLookup("svm-uuid"),
				{method: http.MethodDelete, path: "/api/protocols/san/iscsi/services/svm-uuid", status: http.StatusBadRequest,
					resp: `{"error":{"message":"The iSCSI service must be disabled before it can be deleted.","code":"5374898"}}`},
			},
			wantText: "The iSCSI service must be disabled before it can be deleted.",
			wantErr:  true,
		},
		{
			name:     "modify iSCSI service without SVM is rejected without calling ONTAP",
			tool:     "modify_iscsi_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name:     "modify iSCSI service rejects unknown operation",
			tool:     "modify_iscsi_service",
			args:     modifyService("disable", nil),
			wantText: `unsupported operation "disable"`,
			wantErr:  true,
		},

		// network interfaces used by the iSCSI and igroup suites
		{
			name: "create cluster scope interface with IP in Default ipspace",
			tool: "create_network_ip_interface",
			args: map[string]any{"cluster_name": contractCluster, "name": "cl_mg", "scope": "cluster", "ipspace_name": "Default",
				"ip.address": "10.63.41.6", "ip.netmask": "18", "location.home_node": "umeng-aff300-06"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/network/ip/interfaces", status: http.StatusCreated,
					body: `{"name":"cl_mg","scope":"cluster","ipspace":{"name":"Default"},"ip":{"address":"10.63.41.6","netmask":"18"},
						"location":{"home_node":{"name":"umeng-aff300-06"}}}`},
			},
			wantText: "Network IP interface created successfully",
		},
		{
			name: "create svm scope interface with IP on node",
			tool: "create_network_ip_interface",
			args: map[string]any{"cluster_name": contractCluster, "name": "svg1", "scope": "svm", "svm_name": "marketing",
				"ip.address": "10.63.41.7", "ip.netmask": "18", "location.home_node": "umeng-aff300-06"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/network/ip/interfaces", status: http.StatusCreated,
					body: `{"name":"svg1","scope":"svm","svm":{"name":"marketing"},"ip":{"address":"10.63.41.7","netmask":"18"},
						"location":{"home_node":{"name":"umeng-aff300-06"}}}`},
			},
			wantText: "Network IP interface created successfully",
		},
		{
			// TestIGroupLUNMap creates block-protocol LIFs.
			name: "create svm scope interface with blocks service policy",
			tool: "create_network_ip_interface",
			args: map[string]any{"cluster_name": contractCluster, "name": "svg1", "scope": "svm", "svm_name": "marketing",
				"ip.address": "10.63.41.117", "ip.netmask": "18", "location.home_node": "umeng-aff300-05", "service_policy": "default-data-blocks"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/network/ip/interfaces", status: http.StatusCreated,
					body: `{"name":"svg1","scope":"svm","svm":{"name":"marketing"},"ip":{"address":"10.63.41.117","netmask":"18"},
						"location":{"home_node":{"name":"umeng-aff300-05"}},"service_policy":{"name":"default-data-blocks"}}`},
			},
			wantText: "Network IP interface created successfully",
		},
		{
			name: "create svm scope interface on broadcast domain",
			tool: "create_network_ip_interface",
			args: map[string]any{"cluster_name": contractCluster, "name": "svg1", "scope": "svm", "svm_name": "marketing",
				"ip.address": "10.63.41.7", "ip.netmask": "18", "location.broadcast_domain": "Default"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/network/ip/interfaces", status: http.StatusCreated,
					body: `{"name":"svg1","scope":"svm","svm":{"name":"marketing"},"ip":{"address":"10.63.41.7","netmask":"18"},
						"location":{"broadcast_domain":{"name":"Default"}}}`},
			},
			wantText: "Network IP interface created successfully",
		},
		{
			name: "create interface surfaces ONTAP error",
			tool: "create_network_ip_interface",
			args: map[string]any{"cluster_name": contractCluster, "name": "svg1", "scope": "svm", "svm_name": "marketing",
				"ip.address": "10.63.41.7", "ip.netmask": "18", "location.broadcast_domain": "Default"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/network/ip/interfaces", status: http.StatusBadRequest,
					body: `{"name":"svg1","scope":"svm","svm":{"name":"marketing"},"ip":{"address":"10.63.41.7","netmask":"18"},
						"location":{"broadcast_domain":{"name":"Default"}}}`,
					resp: `{"error":{"message":"Duplicate IP address 10.63.41.7.","code":"1376963"}}`},
			},
			wantText: "Duplicate IP address 10.63.41.7.",
			wantErr:  true,
		},
		{
			name:     "create interface without netmask is rejected without calling ONTAP",
			tool:     "create_network_ip_interface",
			args:     map[string]any{"cluster_name": contractCluster, "name": "svg1", "scope": "svm", "svm_name": "marketing", "ip.address": "10.63.41.7", "location.home_node": "n1"},
			wantText: "network IP address and IP netmask are required",
			wantErr:  true,
		},
		{
			name:     "create interface without node or broadcast domain is rejected without calling ONTAP",
			tool:     "create_network_ip_interface",
			args:     map[string]any{"cluster_name": contractCluster, "name": "svg1", "scope": "svm", "svm_name": "marketing", "ip.address": "10.63.41.7", "ip.netmask": "18"},
			wantText: "home node name OR broadcast domain is required",
			wantErr:  true,
		},
		{
			name:     "create interface without SVM or ipspace is rejected without calling ONTAP",
			tool:     "create_network_ip_interface",
			args:     map[string]any{"cluster_name": contractCluster, "name": "cl_mg", "scope": "cluster", "ip.address": "10.63.41.6", "ip.netmask": "18", "location.home_node": "n1"},
			wantText: "SVM name OR IPSpace name is required",
			wantErr:  true,
		},
		{
			name: "delete cluster scope interface",
			tool: "modify_network_ip_interface",
			args: deleteLIF("cl_mg", "cluster"),
			ontap: []exchange{
				iscsiLIFLookup("cl_mg", "cluster", "", "lif-uuid"),
				{method: http.MethodDelete, path: "/api/network/ip/interfaces/lif-uuid"},
			},
			wantText: "Network IP interface deleted successfully",
		},
		{
			name: "delete svm scope interface filters by SVM",
			tool: "modify_network_ip_interface",
			args: deleteLIF("svg1", "svm"),
			ontap: []exchange{
				iscsiLIFLookup("svg1", "svm", "marketing", "lif-uuid"),
				{method: http.MethodDelete, path: "/api/network/ip/interfaces/lif-uuid"},
			},
			wantText: "Network IP interface deleted successfully",
		},
		{
			// The integration suite's "Clean ... network interface" steps depend on this message.
			name: "delete of missing cluster scope interface",
			tool: "modify_network_ip_interface",
			args: deleteLIF("cl_mg", "cluster"),
			ontap: []exchange{
				iscsiLIFLookup("cl_mg", "cluster", "", ""),
			},
			wantText: "failed to get detail of network interface name cl_mg because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete of missing svm scope interface",
			tool: "modify_network_ip_interface",
			args: deleteLIF("svg1", "svm"),
			ontap: []exchange{
				iscsiLIFLookup("svg1", "svm", "marketing", ""),
			},
			wantText: "failed to get detail of network interface name svg1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete with ambiguous interface name does not delete",
			tool: "modify_network_ip_interface",
			args: deleteLIF("cl_mg", "cluster"),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/network/ip/interfaces?name=cl_mg&scope=cluster",
					resp: `{"num_records":2,"records":[{"uuid":"a"},{"uuid":"b"}]}`},
			},
			wantText: "multiple network interfaces found with name cl_mg",
			wantErr:  true,
		},
		{
			name: "delete interface surfaces ONTAP error",
			tool: "modify_network_ip_interface",
			args: deleteLIF("svg1", "svm"),
			ontap: []exchange{
				iscsiLIFLookup("svg1", "svm", "marketing", "lif-uuid"),
				{method: http.MethodDelete, path: "/api/network/ip/interfaces/lif-uuid", status: http.StatusBadRequest,
					resp: `{"error":{"message":"The LIF is in use by an iSCSI session.","code":"1967082"}}`},
			},
			wantText: "The LIF is in use by an iSCSI session.",
			wantErr:  true,
		},
		{
			name:     "delete svm scope interface without SVM is rejected without calling ONTAP",
			tool:     "modify_network_ip_interface",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "name": "svg1", "scope": "svm"},
			wantText: "SVM name is required",
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
