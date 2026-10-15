package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// nfsServiceLookupSVM is the SVM name-to-UUID lookup that precedes every NFS service update and
// delete; an empty uuid means the SVM does not exist.
func nfsServiceLookupSVM(svm, uuid string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/svm/svms?fields=uuid&name=" + svm, resp: `{"num_records":0,"records":[]}`}
	if uuid != "" {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

func TestNFSServiceContract(t *testing.T) {
	const (
		servicesPath = "/api/protocols/nfs/services"
		servicePath  = servicesPath + "/svm-uuid"
	)

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		// create_nfs_service
		{
			name: "create with NFSv3 enabled and NFSv4.0 disabled",
			tool: "create_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "nfsSvc", "v3_enabled": "true", "v40_enabled": "false"},
			ontap: []exchange{
				{method: http.MethodPost, path: servicesPath, status: http.StatusCreated,
					body: `{"svm":{"name":"nfsSvc"},"enabled":true,"protocol":{"v3_enabled":true,"v40_enabled":false}}`},
			},
			wantText: "NFS service created successfully",
		},
		{
			name: "create defaults to enabled and omits unset protocols",
			tool: "create_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "nfsSvc"},
			ontap: []exchange{
				{method: http.MethodPost, path: servicesPath, status: http.StatusCreated, body: `{"svm":{"name":"nfsSvc"},"enabled":true}`},
			},
			wantText: "NFS service created successfully",
		},
		{
			name: "create disabled with NFSv4.1",
			tool: "create_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "nfsSvc", "enabled": "false", "v41_enabled": "true"},
			ontap: []exchange{
				{method: http.MethodPost, path: servicesPath, status: http.StatusCreated,
					body: `{"svm":{"name":"nfsSvc"},"enabled":false,"protocol":{"v41_enabled":true}}`},
			},
			wantText: "NFS service created successfully",
		},
		{
			name:     "create with invalid boolean is rejected without calling ONTAP",
			tool:     "create_nfs_service",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "nfsSvc", "v40_enabled": "maybe"},
			wantText: `invalid value for v40_enabled: "maybe"`,
			wantErr:  true,
		},
		{
			name:     "create without SVM is rejected without calling ONTAP",
			tool:     "create_nfs_service",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "nfsSvc"},
			ontap: []exchange{
				{method: http.MethodPost, path: servicesPath, status: http.StatusConflict, body: `{"svm":{"name":"nfsSvc"},"enabled":true}`,
					resp: `{"error":{"message":"NFS server already exists for Vserver nfsSvc.","code":"3276916"}}`},
			},
			wantText: "NFS server already exists for Vserver nfsSvc.",
			wantErr:  true,
		},

		// modify_nfs_service
		{
			name: "update enables NFSv4.0 by SVM UUID",
			tool: "modify_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nfsSvc",
				"nfs_service_update": map[string]any{"v40_enabled": "true"}},
			ontap: []exchange{
				nfsServiceLookupSVM("nfsSvc", "svm-uuid"),
				{method: http.MethodPatch, path: servicePath, body: `{"protocol":{"v40_enabled":true}}`},
			},
			wantText: "NFS service updated successfully",
		},
		{
			name: "update disables the service",
			tool: "modify_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nfsSvc",
				"nfs_service_update": map[string]any{"enabled": "false", "v3_enabled": "false"}},
			ontap: []exchange{
				nfsServiceLookupSVM("nfsSvc", "svm-uuid"),
				{method: http.MethodPatch, path: servicePath, body: `{"enabled":false,"protocol":{"v3_enabled":false}}`},
			},
			wantText: "NFS service updated successfully",
		},
		{
			name:     "update with no fields is rejected without calling ONTAP",
			tool:     "modify_nfs_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nfsSvc"},
			wantText: "at least one updatable field must be provided",
			wantErr:  true,
		},
		{
			name: "update with invalid boolean is rejected without calling ONTAP",
			tool: "modify_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nfsSvc",
				"nfs_service_update": map[string]any{"v41_enabled": "yes please"}},
			wantText: `invalid value for v41_enabled "yes please"`,
			wantErr:  true,
		},
		{
			name: "update on missing SVM",
			tool: "modify_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "ghost",
				"nfs_service_update": map[string]any{"v40_enabled": "true"}},
			ontap:    []exchange{nfsServiceLookupSVM("ghost", "")},
			wantText: "failed to get details of SVM ghost because it does not exist",
			wantErr:  true,
		},
		{
			name: "update surfaces ONTAP error message",
			tool: "modify_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nfsSvc",
				"nfs_service_update": map[string]any{"v40_enabled": "true"}},
			ontap: []exchange{
				nfsServiceLookupSVM("nfsSvc", "svm-uuid"),
				{method: http.MethodPatch, path: servicePath, body: `{"protocol":{"v40_enabled":true}}`, status: http.StatusNotFound,
					resp: `{"error":{"message":"entry doesn't exist","code":"4"}}`},
			},
			wantText: "entry doesn't exist",
			wantErr:  true,
		},
		{
			name: "delete by SVM UUID",
			tool: "modify_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "nfsSvc"},
			ontap: []exchange{
				nfsServiceLookupSVM("nfsSvc", "svm-uuid"),
				{method: http.MethodDelete, path: servicePath},
			},
			wantText: "NFS service deleted successfully",
		},
		{
			// The integration suite's "Clean NFS service" step depends on ONTAP's message passing through.
			name: "delete when no NFS service exists",
			tool: "modify_nfs_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "nfsSvc"},
			ontap: []exchange{
				nfsServiceLookupSVM("nfsSvc", "svm-uuid"),
				{method: http.MethodDelete, path: servicePath, status: http.StatusNotFound, resp: `{"error":{"message":"entry doesn't exist","code":"4"}}`},
			},
			wantText: "entry doesn't exist",
			wantErr:  true,
		},
		{
			name:     "delete on missing SVM",
			tool:     "modify_nfs_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "ghost"},
			ontap:    []exchange{nfsServiceLookupSVM("ghost", "")},
			wantText: "failed to get details of SVM ghost because it does not exist",
			wantErr:  true,
		},
		{
			name:     "modify without SVM is rejected without calling ONTAP",
			tool:     "modify_nfs_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_nfs_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "restart", "svm_name": "nfsSvc"},
			wantText: `unsupported operation "restart"`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_nfs_service",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "nfsSvc"},
			wantText: "cluster nope not found",
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
