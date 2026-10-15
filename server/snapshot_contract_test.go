package server

import (
	"net/http"
	"strings"
	"testing"
)

// TestSnapshotContract mirrors the snapshot steps of integration/test/snapshot_test.go: create two
// snapshots, restore the volume from one, and delete it, including the "Clean snapshot" step that
// runs before the snapshot exists. In multiplex mode restore and delete go through modify_snapshot.
func TestSnapshotContract(t *testing.T) {
	const (
		accepted    = `{"job":{"uuid":"job-1"}}`
		volumeUUID  = `{"num_records":1,"records":[{"uuid":"vol-uuid"}]}`
		snapUUID    = `{"num_records":1,"records":[{"uuid":"snap-uuid"}]}`
		empty       = `{"num_records":0,"records":[]}`
		jobFailed   = `{"uuid":"job-1","state":"failure","error":{"message":"Snapshot operation failed: volume busy","code":"1638555"}}`
		volumePath  = "/api/storage/volumes/vol-uuid"
		snapshotsAt = volumePath + "/snapshots"
	)
	lookupVol := func(resp string) exchange {
		return exchange{method: http.MethodGet, path: "/api/storage/volumes?fields=uuid&name=docs&svm.name=marketing", resp: resp}
	}
	lookupSnap := func(name, resp string) exchange {
		return exchange{method: http.MethodGet, path: snapshotsAt + "?fields=uuid&name=" + name, resp: resp}
	}
	job := exchange{method: http.MethodGet, path: "/api/cluster/jobs/job-1", resp: `{"uuid":"job-1","state":"success"}`}
	jobFailure := exchange{method: http.MethodGet, path: "/api/cluster/jobs/job-1", resp: jobFailed}
	snapArgs := func(name string) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "volume_name": "docs", "name": name}
	}
	modifyArgs := func(op, name string) map[string]any {
		args := snapArgs(name)
		args["operation"] = op
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
			name: "create snapshot",
			tool: "create_snapshot",
			args: snapArgs("localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				{method: http.MethodPost, path: snapshotsAt, body: `{"name":"localsnap"}`, status: http.StatusAccepted, resp: accepted},
				job,
			},
			wantText: "Snapshot created successfully",
		},
		{
			name: "create 2nd snapshot",
			tool: "create_snapshot",
			args: snapArgs("recentsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				{method: http.MethodPost, path: snapshotsAt, body: `{"name":"recentsnap"}`, status: http.StatusAccepted, resp: accepted},
				job,
			},
			wantText: "Snapshot created successfully",
		},
		{
			name:     "create on missing volume stops after lookup",
			tool:     "create_snapshot",
			args:     snapArgs("localsnap"),
			ontap:    []exchange{lookupVol(empty)},
			wantText: "volume=docs on svm=marketing does not exist",
			wantErr:  true,
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_snapshot",
			args: snapArgs("localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				{method: http.MethodPost, path: snapshotsAt, body: `{"name":"localsnap"}`, status: http.StatusConflict,
					resp: `{"error":{"message":"Snapshot copy \"localsnap\" of volume \"docs\" already exists.","code":"1638600"}}`},
			},
			wantText: `message: Snapshot copy "localsnap" of volume "docs" already exists. code: 1638600`,
			wantErr:  true,
		},
		{
			name: "create surfaces async job failure",
			tool: "create_snapshot",
			args: snapArgs("localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				{method: http.MethodPost, path: snapshotsAt, body: `{"name":"localsnap"}`, status: http.StatusAccepted, resp: accepted},
				jobFailure,
			},
			wantText: "job failed code=1638555 msg=Snapshot operation failed: volume busy",
			wantErr:  true,
		},
		{
			name:     "create without snapshot name sends nothing",
			tool:     "create_snapshot",
			args:     snapArgs(""),
			wantText: "snapshot name is required",
			wantErr:  true,
		},
		{
			name: "restore volume from snapshot",
			tool: "modify_snapshot",
			args: modifyArgs("restore", "localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				{method: http.MethodPatch, path: volumePath, body: `{"restore_to":{"snapshot":{"name":"localsnap"}}}`, status: http.StatusAccepted, resp: accepted},
				job,
			},
			wantText: "Snapshot restored successfully",
		},
		{
			name:     "restore on missing volume stops after lookup",
			tool:     "modify_snapshot",
			args:     modifyArgs("restore", "localsnap"),
			ontap:    []exchange{lookupVol(empty)},
			wantText: "volume=docs on svm=marketing does not exist",
			wantErr:  true,
		},
		{
			name: "restore surfaces ONTAP error message",
			tool: "modify_snapshot",
			args: modifyArgs("restore", "ghost"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				{method: http.MethodPatch, path: volumePath, body: `{"restore_to":{"snapshot":{"name":"ghost"}}}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Snapshot copy \"ghost\" does not exist.","code":"2"}}`},
			},
			wantText: `message: Snapshot copy "ghost" does not exist. code: 2`,
			wantErr:  true,
		},
		{
			name: "restore surfaces async job failure",
			tool: "modify_snapshot",
			args: modifyArgs("restore", "localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				{method: http.MethodPatch, path: volumePath, body: `{"restore_to":{"snapshot":{"name":"localsnap"}}}`, status: http.StatusAccepted, resp: accepted},
				jobFailure,
			},
			wantText: "job failed code=1638555",
			wantErr:  true,
		},
		{
			name: "delete snapshot by UUID",
			tool: "modify_snapshot",
			args: modifyArgs("delete", "localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				lookupSnap("localsnap", snapUUID),
				{method: http.MethodDelete, path: snapshotsAt + "/snap-uuid", status: http.StatusAccepted, resp: accepted},
				job,
			},
			wantText: "Snapshot deleted successfully",
		},
		{
			// The integration "Clean snapshot" step runs before the snapshot exists and expects "does not exist".
			name: "delete of missing snapshot",
			tool: "modify_snapshot",
			args: modifyArgs("delete", "localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				lookupSnap("localsnap", empty),
			},
			wantText: "snapshot=localsnap on volume uuid=vol-uuid does not exist",
			wantErr:  true,
		},
		{
			name:     "delete on missing volume stops after lookup",
			tool:     "modify_snapshot",
			args:     modifyArgs("delete", "localsnap"),
			ontap:    []exchange{lookupVol(empty)},
			wantText: "volume=docs on svm=marketing does not exist",
			wantErr:  true,
		},
		{
			name: "delete surfaces async job failure",
			tool: "modify_snapshot",
			args: modifyArgs("delete", "localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				lookupSnap("localsnap", snapUUID),
				{method: http.MethodDelete, path: snapshotsAt + "/snap-uuid", status: http.StatusAccepted, resp: accepted},
				jobFailure,
			},
			wantText: "job failed code=1638555 msg=Snapshot operation failed: volume busy",
			wantErr:  true,
		},
		{
			name: "delete surfaces ONTAP error message",
			tool: "modify_snapshot",
			args: modifyArgs("delete", "localsnap"),
			ontap: []exchange{
				lookupVol(volumeUUID),
				lookupSnap("localsnap", snapUUID),
				{method: http.MethodDelete, path: snapshotsAt + "/snap-uuid", status: http.StatusConflict,
					resp: `{"error":{"message":"Snapshot copy is busy.","code":"1638430"}}`},
			},
			wantText: "message: Snapshot copy is busy. code: 1638430",
			wantErr:  true,
		},
		{
			name:     "modify without volume name sends nothing",
			tool:     "modify_snapshot",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "volume_name": "", "name": "localsnap"},
			wantText: "volume name is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_snapshot",
			args:     modifyArgs("rename", "localsnap"),
			wantText: `unsupported operation "rename"; supported values: restore, delete`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_snapshot",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "marketing", "volume_name": "docs", "name": "localsnap"},
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
