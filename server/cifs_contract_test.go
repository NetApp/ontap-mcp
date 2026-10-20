package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// cifsLookupShare is the share lookup that precedes every CIFS share update and delete; an empty
// svmUUID means the share does not exist.
func cifsLookupShare(share, svmUUID string) exchange {
	query := url.Values{"name": {share}, "svm": {"vs_test4"}}
	e := exchange{method: http.MethodGet, path: "/api/protocols/cifs/shares?" + query.Encode(), resp: `{"num_records":0,"records":[]}`}
	if svmUUID != "" {
		e.resp = `{"num_records":1,"records":[{"svm":{"name":"vs_test4","uuid":"` + svmUUID + `"},"name":"` + share + `"}]}`
	}
	return e
}

func TestCIFSContract(t *testing.T) {
	const (
		sharesPath = "/api/protocols/cifs/shares"
		sharePath  = sharesPath + "/svm-uuid/cifsFin"
		// oddShare has characters that must be escaped in a path segment: '#' would otherwise
		// start a fragment and truncate the path to ".../svm-uuid/a".
		oddShare     = "a#b c"
		oddSharePath = sharesPath + "/svm-uuid/a%23b%20c"
	)

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		// create_cifs_share
		{
			name: "create share",
			tool: "create_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "vs_test4", "name": "cifsFin", "path": "/"},
			ontap: []exchange{
				{method: http.MethodPost, path: sharesPath, status: http.StatusCreated, body: `{"svm":{"name":"vs_test4"},"name":"cifsFin","path":"/"}`},
			},
			wantText: "CIFS share created successfully",
		},
		{
			name:     "create without path is rejected without calling ONTAP",
			tool:     "create_cifs_share",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "vs_test4", "name": "cifsFin", "path": ""},
			wantText: "cifs share path is required",
			wantErr:  true,
		},
		{
			name:     "create without SVM is rejected without calling ONTAP",
			tool:     "create_cifs_share",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "", "name": "cifsFin", "path": "/"},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "vs_test4", "name": "cifsFin", "path": "/"},
			ontap: []exchange{
				{method: http.MethodPost, path: sharesPath, status: http.StatusBadRequest, body: `{"svm":{"name":"vs_test4"},"name":"cifsFin","path":"/"}`,
					resp: `{"error":{"message":"Failed to create share cifsFin. Reason: A CIFS server must exist.","code":"655551"}}`},
			},
			wantText: "A CIFS server must exist.",
			wantErr:  true,
		},

		// modify_cifs_share
		{
			name: "update path by SVM UUID and name",
			tool: "modify_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "vs_test4", "name": "cifsFin",
				"cifs_share_update": map[string]any{"path": "/vol_test2"}},
			ontap: []exchange{
				cifsLookupShare("cifsFin", "svm-uuid"),
				{method: http.MethodPatch, path: sharePath, body: `{"path":"/vol_test2"}`, resp: `{}`},
			},
			wantText: "CIFS share updated successfully",
		},
		{
			name: "update escapes share name in path",
			tool: "modify_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "vs_test4", "name": oddShare,
				"cifs_share_update": map[string]any{"path": "/vol_test2"}},
			ontap: []exchange{
				cifsLookupShare(oddShare, "svm-uuid"),
				{method: http.MethodPatch, path: oddSharePath, body: `{"path":"/vol_test2"}`, resp: `{}`},
			},
			wantText: "CIFS share updated successfully",
		},
		{
			name: "update of missing share",
			tool: "modify_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "vs_test4", "name": "ghost",
				"cifs_share_update": map[string]any{"path": "/vol_test2"}},
			ontap:    []exchange{cifsLookupShare("ghost", "")},
			wantText: "failed to get detail of cifs share ghost because it does not exist",
			wantErr:  true,
		},
		{
			name:     "update with no fields is rejected without calling ONTAP",
			tool:     "modify_cifs_share",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "vs_test4", "name": "cifsFin"},
			wantText: "at least one updatable field must be provided: path",
			wantErr:  true,
		},
		{
			name: "update surfaces ONTAP error message",
			tool: "modify_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "vs_test4", "name": "cifsFin",
				"cifs_share_update": map[string]any{"path": "/nowhere"}},
			ontap: []exchange{
				cifsLookupShare("cifsFin", "svm-uuid"),
				{method: http.MethodPatch, path: sharePath, body: `{"path":"/nowhere"}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"The specified path \"/nowhere\" does not exist in the namespace belonging to Vserver \"vs_test4\".","code":"655404"}}`},
			},
			wantText: "does not exist in the namespace",
			wantErr:  true,
		},
		{
			name: "delete by SVM UUID and name",
			tool: "modify_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "vs_test4", "name": "cifsFin"},
			ontap: []exchange{
				cifsLookupShare("cifsFin", "svm-uuid"),
				{method: http.MethodDelete, path: sharePath},
			},
			wantText: "CIFS share deleted successfully",
		},
		{
			name: "delete escapes share name in path",
			tool: "modify_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "vs_test4", "name": oddShare},
			ontap: []exchange{
				cifsLookupShare(oddShare, "svm-uuid"),
				{method: http.MethodDelete, path: oddSharePath},
			},
			wantText: "CIFS share deleted successfully",
		},
		{
			// The integration suite's "Clean CIFS share" steps depend on this message.
			name:     "delete of missing share",
			tool:     "modify_cifs_share",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "vs_test4", "name": "ghost"},
			ontap:    []exchange{cifsLookupShare("ghost", "")},
			wantText: "failed to get detail of cifs share ghost because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete surfaces ONTAP error message",
			tool: "modify_cifs_share",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "vs_test4", "name": "cifsFin"},
			ontap: []exchange{
				cifsLookupShare("cifsFin", "svm-uuid"),
				{method: http.MethodDelete, path: sharePath, status: http.StatusForbidden, resp: `{"error":{"message":"not authorized for that command","code":"6"}}`},
			},
			wantText: "not authorized for that command",
			wantErr:  true,
		},
		{
			name:     "modify without share name is rejected without calling ONTAP",
			tool:     "modify_cifs_share",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "vs_test4", "name": ""},
			wantText: "CIFS share name is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_cifs_share",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "rename", "svm_name": "vs_test4", "name": "cifsFin"},
			wantText: `unsupported operation "rename"`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_cifs_share",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "vs_test4", "name": "cifsFin"},
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
