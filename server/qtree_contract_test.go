package server

import (
	"net/http"
	"strings"
	"testing"
)

// TestQtreeContract mirrors the qtree steps of integration/test/qtree_test.go: create a qtree,
// rename it, and delete it, including the "because it does not exist" message the clean step
// accepts. In multiplex mode rename and delete go through modify_qtree.
func TestQtreeContract(t *testing.T) {
	const (
		accepted = `{"job":{"uuid":"job-1"}}`
		qtreeRec = `{"num_records":1,"records":[{"id":1,"volume":{"uuid":"vol-uuid"}}]}`
		empty    = `{"num_records":0,"records":[]}`
		qtreeAt  = "/api/storage/qtrees/vol-uuid/1"
		create   = `{"svm":{"name":"marketing"},"volume":{"name":"docs"},"name":"staff"}`
	)
	lookupQtree := func(name, resp string) exchange {
		return exchange{method: http.MethodGet, path: "/api/storage/qtrees?fields=id,volume.uuid&name=" + name + "&svm=marketing&volume=docs", resp: resp}
	}
	job := exchange{method: http.MethodGet, path: "/api/cluster/jobs/job-1", resp: `{"uuid":"job-1","state":"success"}`}
	jobFailure := exchange{method: http.MethodGet, path: "/api/cluster/jobs/job-1",
		resp: `{"uuid":"job-1","state":"failure","error":{"message":"Qtree operation failed: volume is offline","code":"918235"}}`}
	createArgs := func(name string) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "name": name}
	}
	modifyArgs := func(op, name string, update map[string]any) map[string]any {
		args := createArgs(name)
		args["operation"] = op
		if update != nil {
			args["qtree_update"] = update
		}
		return args
	}
	rename := map[string]any{"new_name": "pay"}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			name: "create qtree staff",
			tool: "create_qtree",
			args: createArgs("staff"),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/qtrees", body: create, status: http.StatusAccepted, resp: accepted},
				job,
			},
			wantText: "Qtree created successfully",
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_qtree",
			args: createArgs("staff"),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/qtrees", body: create, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Volume \"docs\" does not exist in SVM \"marketing\".","code":"917927"}}`},
			},
			wantText: `message: Volume "docs" does not exist in SVM "marketing". code: 917927`,
			wantErr:  true,
		},
		{
			name: "create surfaces async job failure",
			tool: "create_qtree",
			args: createArgs("staff"),
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/storage/qtrees", body: create, status: http.StatusAccepted, resp: accepted},
				jobFailure,
			},
			wantText: "job failed code=918235 msg=Qtree operation failed: volume is offline",
			wantErr:  true,
		},
		{
			name:     "create without qtree name sends nothing",
			tool:     "create_qtree",
			args:     createArgs(""),
			wantText: "qtree name is required",
			wantErr:  true,
		},
		{
			name: "rename qtree staff to pay",
			tool: "modify_qtree",
			args: modifyArgs("update", "staff", rename),
			ontap: []exchange{
				lookupQtree("staff", qtreeRec),
				{method: http.MethodPatch, path: qtreeAt, body: `{"name":"pay"}`, status: http.StatusAccepted, resp: accepted},
				job,
			},
			wantText: "Qtree updated successfully",
		},
		{
			name:     "rename of missing qtree",
			tool:     "modify_qtree",
			args:     modifyArgs("update", "staff", rename),
			ontap:    []exchange{lookupQtree("staff", empty)},
			wantText: "failed to get detail of qtree staff because it does not exist",
			wantErr:  true,
		},
		{
			name: "rename surfaces async job failure",
			tool: "modify_qtree",
			args: modifyArgs("update", "staff", rename),
			ontap: []exchange{
				lookupQtree("staff", qtreeRec),
				{method: http.MethodPatch, path: qtreeAt, body: `{"name":"pay"}`, status: http.StatusAccepted, resp: accepted},
				jobFailure,
			},
			wantText: "job failed code=918235",
			wantErr:  true,
		},
		{
			name:     "rename without new name sends nothing",
			tool:     "modify_qtree",
			args:     modifyArgs("update", "staff", nil),
			wantText: "at least one updatable field must be provided: new_name",
			wantErr:  true,
		},
		{
			name: "delete qtree pay",
			tool: "modify_qtree",
			args: modifyArgs("delete", "pay", nil),
			ontap: []exchange{
				lookupQtree("pay", qtreeRec),
				{method: http.MethodDelete, path: qtreeAt, status: http.StatusAccepted, resp: accepted},
				job,
			},
			wantText: "Qtree deleted successfully",
		},
		{
			// The integration "Clean qtree" step accepts this message.
			name:     "delete of missing qtree",
			tool:     "modify_qtree",
			args:     modifyArgs("delete", "pay", nil),
			ontap:    []exchange{lookupQtree("pay", empty)},
			wantText: "failed to get detail of qtree pay because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete surfaces ONTAP error message",
			tool: "modify_qtree",
			args: modifyArgs("delete", "pay", nil),
			ontap: []exchange{
				lookupQtree("pay", qtreeRec),
				{method: http.MethodDelete, path: qtreeAt, status: http.StatusConflict,
					resp: `{"error":{"message":"Qtree \"pay\" is not empty.","code":"918236"}}`},
			},
			wantText: `message: Qtree "pay" is not empty. code: 918236`,
			wantErr:  true,
		},
		{
			name:     "modify without volume name sends nothing",
			tool:     "modify_qtree",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "volume_name": "", "name": "pay"},
			wantText: "volume name is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_qtree",
			args:     modifyArgs("rename", "staff", rename),
			wantText: `unsupported operation "rename"; supported values: update, delete`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_qtree",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "marketing", "volume_name": "docs", "name": "pay"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			if isErr != tt.wantErr {
				t.Errorf("isError = %v, want %v (text %q)", isErr, tt.wantErr, text)
			}
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("text = %q, want it to contain %q", text, tt.wantText)
			}
		})
	}
}
