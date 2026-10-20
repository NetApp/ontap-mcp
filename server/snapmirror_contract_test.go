package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestSnapMirrorContract(t *testing.T) {
	const (
		srcPath  = "srsvm:srvol"
		dstPath  = "dtsvm:dtvol"
		smUUID   = "sm-uuid"
		accepted = `{"job":{"uuid":"job-1"}}`
	)
	// lookup is the destination-path lookup that precedes every update and delete. An empty uuid
	// means the relationship does not exist.
	lookup := func(uuid, policyType string) exchange {
		e := exchange{method: http.MethodGet, path: "/api/snapmirror/relationships?fields=uuid,policy.type&destination.path=" + dstPath,
			resp: `{"num_records":0,"records":[]}`}
		if uuid != "" {
			e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `","policy":{"type":"` + policyType + `"}}]}`
		}
		return e
	}
	// patch expects a PATCH of the relationship that ONTAP accepts as job-1.
	patch := func(body string) exchange {
		return exchange{method: http.MethodPatch, path: "/api/snapmirror/relationships/" + smUUID, body: body, status: http.StatusAccepted, resp: accepted}
	}
	jobDone := exchange{method: http.MethodGet, path: "/api/cluster/jobs/job-1", resp: `{"uuid":"job-1","state":"success"}`}
	jobFailed := func(msg, code string) exchange {
		return exchange{method: http.MethodGet, path: "/api/cluster/jobs/job-1",
			resp: `{"uuid":"job-1","state":"failure","error":{"message":"` + msg + `","code":"` + code + `"}}`}
	}
	found := lookup(smUUID, "async")
	missing := lookup("", "")
	modify := func(operation string, update map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "operation": operation, "destination.path": dstPath}
		if update != nil {
			args["snapmirror_update"] = update
		}
		return args
	}
	createArgs := map[string]any{"cluster_name": contractCluster, "source.path": srcPath, "destination.path": dstPath, "policy_name": "Asynchronous"}
	createBody := `{"source":{"path":"srsvm:srvol"},"destination":{"path":"dtsvm:dtvol"},"policy":{"name":"Asynchronous"}}`
	transferArgs := map[string]any{"cluster_name": contractCluster, "destination.path": dstPath}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		// Create SnapMirror relationship
		{
			name: "create posts source, destination and policy",
			tool: "create_snapmirror",
			args: createArgs,
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/snapmirror/relationships", body: createBody, status: http.StatusAccepted, resp: accepted},
				jobDone,
			},
			wantText: "SnapMirror relationship created successfully",
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_snapmirror",
			args: createArgs,
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/snapmirror/relationships", body: createBody, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Destination volume \"dtvol\" must be of type DP.","code":"13303812"}}`},
			},
			wantText: `message: Destination volume "dtvol" must be of type DP. code: 13303812`,
			wantErr:  true,
		},
		{
			name: "create surfaces async job failure",
			tool: "create_snapmirror",
			args: createArgs,
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/snapmirror/relationships", body: createBody, status: http.StatusAccepted, resp: accepted},
				jobFailed("Policy Asynchronous not found", "13303825"),
			},
			wantText: "job failed code=13303825 msg=Policy Asynchronous not found",
			wantErr:  true,
		},
		{
			name:     "create without source path sends nothing",
			tool:     "create_snapmirror",
			args:     map[string]any{"cluster_name": contractCluster, "source.path": "", "destination.path": dstPath, "policy_name": "Asynchronous"},
			wantText: "source.path is required",
			wantErr:  true,
		},
		{
			name:     "create without destination path sends nothing",
			tool:     "create_snapmirror",
			args:     map[string]any{"cluster_name": contractCluster, "source.path": srcPath, "destination.path": "", "policy_name": "Asynchronous"},
			wantText: "destination.path is required",
			wantErr:  true,
		},
		{
			name:     "create without policy sends nothing",
			tool:     "create_snapmirror",
			args:     map[string]any{"cluster_name": contractCluster, "source.path": srcPath, "destination.path": dstPath, "policy_name": ""},
			wantText: "policy_name is required",
			wantErr:  true,
		},
		{
			name:     "create on unknown cluster sends nothing",
			tool:     "create_snapmirror",
			args:     map[string]any{"cluster_name": "nope", "source.path": srcPath, "destination.path": dstPath, "policy_name": "Asynchronous"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},

		// Update SnapMirror relationship (transfer schedule)
		{
			name:     "modify sets transfer schedule by UUID",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"transfer_schedule_name": "hourly"}),
			ontap:    []exchange{found, patch(`{"transfer_schedule":{"name":"hourly"}}`), jobDone},
			wantText: "SnapMirror relationship updated successfully",
		},
		// Update SnapMirror relationship 2 (policy)
		{
			name:     "modify changes policy by UUID",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"policy_name": "MirrorAndVault"}),
			ontap:    []exchange{found, patch(`{"policy":{"name":"MirrorAndVault"}}`), jobDone},
			wantText: "SnapMirror relationship updated successfully",
		},
		{
			name:     "modify of missing relationship stops after lookup",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"policy_name": "MirrorAndVault"}),
			ontap:    []exchange{missing},
			wantText: `SnapMirror relationship with destination "dtsvm:dtvol" not found`,
			wantErr:  true,
		},
		{
			name: "modify refuses ambiguous destination",
			tool: "modify_snapmirror",
			args: modify("update", map[string]any{"policy_name": "MirrorAndVault"}),
			ontap: []exchange{{method: http.MethodGet, path: "/api/snapmirror/relationships?fields=uuid,policy.type&destination.path=" + dstPath,
				resp: `{"num_records":2,"records":[{"uuid":"a"},{"uuid":"b"}]}`}},
			wantText: `found 2 SnapMirror relationships with destination "dtsvm:dtvol", expected 1`,
			wantErr:  true,
		},
		{
			name: "modify surfaces ONTAP error on PATCH",
			tool: "modify_snapmirror",
			args: modify("update", map[string]any{"policy_name": "NoSuchPolicy"}),
			ontap: []exchange{
				found,
				{method: http.MethodPatch, path: "/api/snapmirror/relationships/" + smUUID, body: `{"policy":{"name":"NoSuchPolicy"}}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Policy \"NoSuchPolicy\" does not exist.","code":"13303845"}}`},
			},
			wantText: `message: Policy "NoSuchPolicy" does not exist. code: 13303845`,
			wantErr:  true,
		},
		{
			name:     "modify update without fields sends nothing",
			tool:     "modify_snapmirror",
			args:     modify("update", nil),
			wantText: "at least one updatable field must be provided",
			wantErr:  true,
		},
		{
			name:     "modify without destination path sends nothing",
			tool:     "modify_snapmirror",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "destination.path": "", "snapmirror_update": map[string]any{"policy_name": "MirrorAndVault"}},
			wantText: "destination.path is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_snapmirror",
			args:     modify("create", nil),
			wantText: `unsupported operation "create"; supported values: update, delete`,
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown snapmirror_operation",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "abort", "policy_name": "MirrorAndVault"}),
			wantText: `unsupported snapmirror_operation "abort"; supported values: initialize, break, resync, pause, quiesce, resume`,
			wantErr:  true,
		},

		// Initialize SnapMirror relationship
		{
			name:     "initialize patches state snapmirrored",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "initialize"}),
			ontap:    []exchange{found, patch(`{"state":"snapmirrored"}`), jobDone},
			wantText: "SnapMirror relationship initialized successfully",
		},
		{
			name:     "initialize of sync relationship patches state in_sync",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "initialize"}),
			ontap:    []exchange{lookup(smUUID, "sync"), patch(`{"state":"in_sync"}`), jobDone},
			wantText: "SnapMirror relationship initialized successfully",
		},
		{
			name:     "initialize surfaces async job failure",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "initialize"}),
			ontap:    []exchange{found, patch(`{"state":"snapmirrored"}`), jobFailed("Transfer failed: source volume is offline", "13303890")},
			wantText: "job failed code=13303890 msg=Transfer failed: source volume is offline",
			wantErr:  true,
		},
		{
			name:     "initialize of missing relationship stops after lookup",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "initialize"}),
			ontap:    []exchange{missing},
			wantText: `SnapMirror relationship with destination "dtsvm:dtvol" not found`,
			wantErr:  true,
		},

		// Break SnapMirror relationship
		{
			name:     "break patches state broken_off",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "break"}),
			ontap:    []exchange{found, patch(`{"state":"broken_off"}`), jobDone},
			wantText: "SnapMirror relationship broken successfully",
		},
		{
			name:     "break of sync relationship still patches broken_off",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "break"}),
			ontap:    []exchange{lookup(smUUID, "sync"), patch(`{"state":"broken_off"}`), jobDone},
			wantText: "SnapMirror relationship broken successfully",
		},

		// Resync SnapMirror relationship
		{
			name:     "resync patches state snapmirrored",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "resync"}),
			ontap:    []exchange{found, patch(`{"state":"snapmirrored"}`), jobDone},
			wantText: "SnapMirror relationship resynced successfully",
		},

		// Pause SnapMirror relationship
		{
			name:     "pause patches state paused",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "pause"}),
			ontap:    []exchange{found, patch(`{"state":"paused"}`), jobDone},
			wantText: "SnapMirror relationship paused successfully",
		},
		{
			name:     "quiesce is an alias for pause",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "quiesce"}),
			ontap:    []exchange{found, patch(`{"state":"paused"}`), jobDone},
			wantText: "SnapMirror relationship paused successfully",
		},
		{
			name: "pause surfaces ONTAP error on PATCH",
			tool: "modify_snapmirror",
			args: modify("update", map[string]any{"snapmirror_operation": "pause"}),
			ontap: []exchange{
				found,
				{method: http.MethodPatch, path: "/api/snapmirror/relationships/" + smUUID, body: `{"state":"paused"}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Invalid state transition from uninitialized to paused.","code":"13303808"}}`},
			},
			wantText: "message: Invalid state transition from uninitialized to paused. code: 13303808",
			wantErr:  true,
		},

		// Resume SnapMirror relationship
		{
			name:     "resume patches state snapmirrored",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "resume"}),
			ontap:    []exchange{found, patch(`{"state":"snapmirrored"}`), jobDone},
			wantText: "SnapMirror relationship resumed successfully",
		},
		{
			name:     "resume of sync relationship patches state in_sync",
			tool:     "modify_snapmirror",
			args:     modify("update", map[string]any{"snapmirror_operation": "resume"}),
			ontap:    []exchange{lookup(smUUID, "sync"), patch(`{"state":"in_sync"}`), jobDone},
			wantText: "SnapMirror relationship resumed successfully",
		},

		// Delete SnapMirror relationship
		{
			name: "delete removes relationship by UUID",
			tool: "modify_snapmirror",
			args: modify("delete", nil),
			ontap: []exchange{
				found,
				{method: http.MethodDelete, path: "/api/snapmirror/relationships/" + smUUID, status: http.StatusAccepted, resp: accepted},
				jobDone,
			},
			wantText: "SnapMirror relationship deleted successfully",
		},
		{
			name:     "delete of missing relationship stops after lookup",
			tool:     "modify_snapmirror",
			args:     modify("delete", nil),
			ontap:    []exchange{missing},
			wantText: `SnapMirror relationship with destination "dtsvm:dtvol" not found`,
			wantErr:  true,
		},
		{
			name: "delete surfaces async job failure",
			tool: "modify_snapmirror",
			args: modify("delete", nil),
			ontap: []exchange{
				found,
				{method: http.MethodDelete, path: "/api/snapmirror/relationships/" + smUUID, status: http.StatusAccepted, resp: accepted},
				jobFailed("Relationship is busy with a transfer.", "13303869"),
			},
			wantText: "job failed code=13303869 msg=Relationship is busy with a transfer.",
			wantErr:  true,
		},
		{
			name: "delete surfaces ONTAP error",
			tool: "modify_snapmirror",
			args: modify("delete", nil),
			ontap: []exchange{
				found,
				{method: http.MethodDelete, path: "/api/snapmirror/relationships/" + smUUID, status: http.StatusForbidden,
					resp: `{"error":{"message":"not authorized for that command","code":"6"}}`},
			},
			wantText: "message: not authorized for that command code: 6",
			wantErr:  true,
		},

		// Transfer tools: not in the integration test, but they share the lookup.
		{
			name: "update transfer starts a transfer",
			tool: "update_snapmirror_transfer",
			args: transferArgs,
			ontap: []exchange{
				found,
				{method: http.MethodPost, path: "/api/snapmirror/relationships/" + smUUID + "/transfers", body: `{}`, status: http.StatusCreated, resp: `{}`},
			},
			wantText: "SnapMirror transfer updated successfully",
		},
		{
			name:     "update transfer of missing relationship stops after lookup",
			tool:     "update_snapmirror_transfer",
			args:     transferArgs,
			ontap:    []exchange{missing},
			wantText: `SnapMirror relationship with destination "dtsvm:dtvol" not found`,
			wantErr:  true,
		},
		{
			name: "abort transfer aborts the active transfer",
			tool: "abort_snapmirror_transfer",
			args: transferArgs,
			ontap: []exchange{
				found,
				{method: http.MethodGet, path: "/api/snapmirror/relationships/" + smUUID + "/transfers?fields=uuid&state=transferring",
					resp: `{"num_records":1,"records":[{"uuid":"xfer-1"}]}`},
				{method: http.MethodPatch, path: "/api/snapmirror/relationships/" + smUUID + "/transfers/xfer-1", body: `{"state":"aborted"}`, status: http.StatusAccepted, resp: accepted},
				jobDone,
			},
			wantText: "SnapMirror transfer aborted successfully",
		},
		{
			name: "abort transfer aborts every active transfer",
			tool: "abort_snapmirror_transfer",
			args: transferArgs,
			ontap: []exchange{
				found,
				{method: http.MethodGet, path: "/api/snapmirror/relationships/" + smUUID + "/transfers?fields=uuid&state=transferring",
					resp: `{"num_records":2,"records":[{"uuid":"xfer-1"},{"uuid":"xfer-2"}]}`},
				{method: http.MethodPatch, path: "/api/snapmirror/relationships/" + smUUID + "/transfers/xfer-1", body: `{"state":"aborted"}`, status: http.StatusAccepted, resp: accepted},
				jobDone,
				{method: http.MethodPatch, path: "/api/snapmirror/relationships/" + smUUID + "/transfers/xfer-2", body: `{"state":"aborted"}`, status: http.StatusAccepted, resp: `{"job":{"uuid":"job-2"}}`},
				{method: http.MethodGet, path: "/api/cluster/jobs/job-2", resp: `{"uuid":"job-2","state":"success"}`},
			},
			wantText: "SnapMirror transfer aborted successfully",
		},
		{
			name: "abort transfer with nothing transferring",
			tool: "abort_snapmirror_transfer",
			args: transferArgs,
			ontap: []exchange{
				found,
				{method: http.MethodGet, path: "/api/snapmirror/relationships/" + smUUID + "/transfers?fields=uuid&state=transferring",
					resp: `{"num_records":0,"records":[]}`},
			},
			wantText: "SnapMirror transfer with state transferring not found",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			// assert.Contains is not on main yet, so check the substring here.
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("tool output %q does not contain %q", text, tt.wantText)
			}
		})
	}
}
