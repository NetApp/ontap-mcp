package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// svmLookup is the name-to-UUID lookup that precedes every SVM update and delete; an empty uuid
// means the SVM does not exist.
func svmLookup(svm, uuid string) exchange {
	e := exchange{method: http.MethodGet, path: "/api/svm/svms?fields=uuid&name=" + svm, resp: `{"num_records":0,"records":[]}`}
	if uuid != "" {
		e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
	}
	return e
}

// TestSVMContract mirrors integration/test/svm_test.go: clean, create, rename, update state and
// comment, and delete an SVM.
func TestSVMContract(t *testing.T) {
	const (
		svmJobAccepted = `{"job":{"uuid":"svm-job"}}`
		jobPath        = "/api/cluster/jobs/svm-job"
	)
	jobDone := exchange{method: http.MethodGet, path: jobPath, resp: `{"uuid":"svm-job","state":"success"}`}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			// The integration "Clean SVM" step expects this message when the SVM is absent.
			name:     "delete of missing svm",
			tool:     "modify_svm",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing"},
			ontap:    []exchange{svmLookup("marketing", "")},
			wantText: "failed to get details of SVM marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "create svm",
			tool: "create_svm",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/svm/svms", body: `{"name":"marketing"}`, status: http.StatusAccepted, resp: svmJobAccepted},
				jobDone,
			},
			wantText: "SVM created successfully",
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_svm",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/svm/svms", body: `{"name":"marketing"}`, status: http.StatusConflict,
					resp: `{"error":{"message":"Duplicate SVM name marketing.","code":"13434908"}}`},
			},
			wantText: "Duplicate SVM name marketing.",
			wantErr:  true,
		},
		{
			name: "create surfaces async job failure",
			tool: "create_svm",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing"},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/svm/svms", body: `{"name":"marketing"}`, status: http.StatusAccepted, resp: svmJobAccepted},
				{method: http.MethodGet, path: jobPath,
					resp: `{"uuid":"svm-job","state":"failure","error":{"message":"No aggregates available for root volume","code":"13434880"}}`},
			},
			wantText: "No aggregates available for root volume",
			wantErr:  true,
		},
		{
			name:     "create without name is rejected without calling ONTAP",
			tool:     "create_svm",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name: "rename svm",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing",
				"svm_update": map[string]any{"new_name": "marketingNew"}},
			ontap: []exchange{
				svmLookup("marketing", "svm-uuid"),
				{method: http.MethodPatch, path: "/api/svm/svms/svm-uuid", body: `{"name":"marketingNew"}`, status: http.StatusAccepted, resp: svmJobAccepted},
				jobDone,
			},
			wantText: "SVM updated successfully",
		},
		{
			name: "update state and comment",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketingNew",
				"svm_update": map[string]any{"state": "stopped", "comment": "stop_svm"}},
			ontap: []exchange{
				svmLookup("marketingNew", "svm-uuid"),
				{method: http.MethodPatch, path: "/api/svm/svms/svm-uuid", body: `{"state":"stopped","comment":"stop_svm"}`, status: http.StatusAccepted, resp: svmJobAccepted},
				jobDone,
			},
			wantText: "SVM updated successfully",
		},
		{
			name: "update completed synchronously",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing",
				"svm_update": map[string]any{"comment": "hello"}},
			ontap: []exchange{
				svmLookup("marketing", "svm-uuid"),
				{method: http.MethodPatch, path: "/api/svm/svms/svm-uuid", body: `{"comment":"hello"}`},
			},
			wantText: "SVM updated successfully",
		},
		{
			name: "update clears comment with empty string",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing",
				"svm_update": map[string]any{"comment": ""}},
			ontap: []exchange{
				svmLookup("marketing", "svm-uuid"),
				{method: http.MethodPatch, path: "/api/svm/svms/svm-uuid", body: `{"comment":""}`},
			},
			wantText: "SVM updated successfully",
		},
		{
			name: "update without comment leaves it out of the body",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing",
				"svm_update": map[string]any{"new_name": "marketingNew", "state": "running"}},
			ontap: []exchange{
				svmLookup("marketing", "svm-uuid"),
				{method: http.MethodPatch, path: "/api/svm/svms/svm-uuid", body: `{"name":"marketingNew","state":"running"}`},
			},
			wantText: "SVM updated successfully",
		},
		{
			name: "update surfaces async job failure",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing",
				"svm_update": map[string]any{"state": "stopped"}},
			ontap: []exchange{
				svmLookup("marketing", "svm-uuid"),
				{method: http.MethodPatch, path: "/api/svm/svms/svm-uuid", body: `{"state":"stopped"}`, status: http.StatusAccepted, resp: svmJobAccepted},
				{method: http.MethodGet, path: jobPath,
					resp: `{"uuid":"svm-job","state":"failure","error":{"message":"Failed to stop Vserver","code":"13434881"}}`},
			},
			wantText: "Failed to stop Vserver",
			wantErr:  true,
		},
		{
			name: "update surfaces ONTAP error message",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing",
				"svm_update": map[string]any{"new_name": "taken"}},
			ontap: []exchange{
				svmLookup("marketing", "svm-uuid"),
				{method: http.MethodPatch, path: "/api/svm/svms/svm-uuid", body: `{"name":"taken"}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Duplicate SVM name taken.","code":"13434908"}}`},
			},
			wantText: "Duplicate SVM name taken.",
			wantErr:  true,
		},
		{
			name: "update of missing svm",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "ghost",
				"svm_update": map[string]any{"state": "stopped"}},
			ontap:    []exchange{svmLookup("ghost", "")},
			wantText: "failed to get details of SVM ghost because it does not exist",
			wantErr:  true,
		},
		{
			name:     "update with no fields is rejected without calling ONTAP",
			tool:     "modify_svm",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing"},
			wantText: "at least one updatable field must be provided: new_name, comment, or state",
			wantErr:  true,
		},
		{
			name: "delete svm",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketingNew"},
			ontap: []exchange{
				svmLookup("marketingNew", "svm-uuid"),
				{method: http.MethodDelete, path: "/api/svm/svms/svm-uuid", status: http.StatusAccepted, resp: svmJobAccepted},
				jobDone,
			},
			wantText: "SVM deleted successfully",
		},
		{
			name: "delete surfaces async job failure",
			tool: "modify_svm",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing"},
			ontap: []exchange{
				svmLookup("marketing", "svm-uuid"),
				{method: http.MethodDelete, path: "/api/svm/svms/svm-uuid", status: http.StatusAccepted, resp: svmJobAccepted},
				{method: http.MethodGet, path: jobPath,
					resp: `{"uuid":"svm-job","state":"failure","error":{"message":"Cannot delete Vserver because it has volumes","code":"13434882"}}`},
			},
			wantText: "Cannot delete Vserver because it has volumes",
			wantErr:  true,
		},
		{
			name:     "modify without name is rejected without calling ONTAP",
			tool:     "modify_svm",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_svm",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "rename", "svm_name": "marketing"},
			wantText: `unsupported operation "rename"`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "create_svm",
			args:     map[string]any{"cluster_name": "nope", "svm_name": "marketing"},
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
				t.Errorf("tool output %q does not contain %q", text, tt.wantText)
			}
		})
	}
}
