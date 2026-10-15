package server

import (
	"net/http"
	"strings"
	"testing"
)

// TestSnapshotPolicyContract mirrors integration/test/snapshot_policy_test.go: create the cron
// schedules the prompts ask for, create two policies, add, update, and remove a schedule entry,
// disable a policy with a comment, and delete the policies, including the "Clean snapshot policy"
// steps that run before the policies exist.
func TestSnapshotPolicyContract(t *testing.T) {
	const (
		policies    = "/api/storage/snapshot-policies"
		policyAt    = policies + "/pol-uuid"
		entries     = policyAt + "/schedules"
		entryAt     = entries + "/sched-uuid"
		schedules   = "/api/cluster/schedules"
		policyUUID  = `{"num_records":1,"records":[{"uuid":"pol-uuid"}]}`
		entryUUID   = `{"num_records":1,"records":[{"schedule":{"uuid":"sched-uuid","name":"2hours"}}]}`
		empty       = `{"num_records":0,"records":[]}`
		policyGone  = "failed to get snapshot policy=every4hours on svm=marketing because it does not exist"
		entryAbsent = "schedule=2hours does not exist in snapshot policy uuid=pol-uuid"
	)
	lookupPolicy := func(name, resp string) exchange {
		return exchange{method: http.MethodGet, path: policies + "?fields=uuid&name=" + name + "&svm.name=marketing", resp: resp}
	}
	lookupEntry := func(resp string) exchange {
		return exchange{method: http.MethodGet, path: entries + "?fields=schedule.uuid,schedule.name&schedule.name=2hours", resp: resp}
	}
	countSchedules := func(name, resp string) exchange {
		return exchange{method: http.MethodGet, path: schedules + "?fields=name&name=" + name + "&return_records=false", resp: resp}
	}
	scheduleArgs := func(name, cron string) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "name": name, "cron_expression": cron}
	}
	policyArgs := func(name, schedule string, count int) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": name, "schedule": schedule, "count": count}
	}
	modifyPolicyArgs := func(op string, update map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "operation": op, "svm_name": "marketing", "name": "every4hours"}
		if update != nil {
			args["snapshot_policy_update"] = update
		}
		return args
	}
	entryArgs := func(op string, update map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "operation": op, "svm_name": "marketing", "policy_name": "every4hours", "schedule_name": "2hours"}
		if update != nil {
			args["snapshot_policy_schedule_update"] = update
		}
		return args
	}
	addEntryArgs := map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "policy_name": "every4hours", "schedule_name": "2hours", "count": 6}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		// create_schedule
		{
			name: "create schedule 4hours",
			tool: "create_schedule",
			args: scheduleArgs("4hours", "0 0,4,8,12,16,20 * * *"),
			ontap: []exchange{
				{method: http.MethodPost, path: schedules, status: http.StatusCreated,
					body: `{"name":"4hours","cron":{"minutes":[0],"hours":[0,4,8,12,16,20],"days":[],"months":[],"weekdays":[]}}`},
			},
			wantText: "Schedule created successfully",
		},
		{
			name: "create schedule 5minutes",
			tool: "create_schedule",
			args: scheduleArgs("5minutes", "0,5,10,15,20,25,30,35,40,45,50,55 * * * *"),
			ontap: []exchange{
				{method: http.MethodPost, path: schedules, status: http.StatusCreated,
					body: `{"name":"5minutes","cron":{"minutes":[0,5,10,15,20,25,30,35,40,45,50,55],"hours":[],"days":[],"months":[],"weekdays":[]}}`},
			},
			wantText: "Schedule created successfully",
		},
		{
			name: "create schedule 2hours",
			tool: "create_schedule",
			args: scheduleArgs("2hours", "0 0,2,4,6,8,10,12,14,16,18,20,22 * * *"),
			ontap: []exchange{
				{method: http.MethodPost, path: schedules, status: http.StatusCreated,
					body: `{"name":"2hours","cron":{"minutes":[0],"hours":[0,2,4,6,8,10,12,14,16,18,20,22],"days":[],"months":[],"weekdays":[]}}`},
			},
			wantText: "Schedule created successfully",
		},
		{
			name: "create schedule surfaces ONTAP error message",
			tool: "create_schedule",
			args: scheduleArgs("4hours", "0 0,4,8,12,16,20 * * *"),
			ontap: []exchange{
				{method: http.MethodPost, path: schedules, status: http.StatusConflict,
					body: `{"name":"4hours","cron":{"minutes":[0],"hours":[0,4,8,12,16,20],"days":[],"months":[],"weekdays":[]}}`,
					resp: `{"error":{"message":"duplicate entry","code":"1"}}`},
			},
			wantText: "message: duplicate entry code: 1",
			wantErr:  true,
		},
		{
			name:     "create schedule rejects step syntax without calling ONTAP",
			tool:     "create_schedule",
			args:     scheduleArgs("5minutes", "*/5 * * * *"),
			wantText: "wrong cron format */5 detected",
			wantErr:  true,
		},
		{
			name: "create schedule at every field's upper bound",
			tool: "create_schedule",
			args: scheduleArgs("edge", "59 23 31 12 6"),
			ontap: []exchange{
				{method: http.MethodPost, path: schedules, status: http.StatusCreated,
					body: `{"name":"edge","cron":{"minutes":[59],"hours":[23],"days":[31],"months":[12],"weekdays":[6]}}`},
			},
			wantText: "Schedule created successfully",
		},
		{
			// Previously the out-of-range minute was dropped and ONTAP got "minutes":[], every minute of hour 0.
			name:     "create schedule rejects out-of-range minute without calling ONTAP",
			tool:     "create_schedule",
			args:     scheduleArgs("midnight", "60 0 * * *"),
			wantText: `invalid cron minutes value "60": must be between 0 and 59`,
			wantErr:  true,
		},
		{
			// Previously Sscanf read "4hours" as 4 and ONTAP got "hours":[4].
			name:     "create schedule rejects trailing garbage without calling ONTAP",
			tool:     "create_schedule",
			args:     scheduleArgs("4hours", "0 4hours * * *"),
			wantText: `invalid cron hours value "4hours": expected *, a number, or a range such as 1-5`,
			wantErr:  true,
		},
		{
			// Previously the sixth field was ignored with a Println to stdout.
			name:     "create schedule rejects six fields without calling ONTAP",
			tool:     "create_schedule",
			args:     scheduleArgs("4hours", "0 4 * * * 2026"),
			wantText: `invalid cron expression "0 4 * * * 2026": expected 5 fields (minutes hours days months weekdays), got 6`,
			wantErr:  true,
		},
		{
			name:     "create schedule without cron sends nothing",
			tool:     "create_schedule",
			args:     scheduleArgs("4hours", ""),
			wantText: "schedule cron expression is required",
			wantErr:  true,
		},

		// create_snapshot_policy
		{
			name: "create policy every4hours with existing schedule",
			tool: "create_snapshot_policy",
			args: policyArgs("every4hours", "4hours", 5),
			ontap: []exchange{
				countSchedules("4hours", `{"num_records":1}`),
				{method: http.MethodPost, path: policies, status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"every4hours","copies":[{"count":5,"schedule":{"name":"4hours"}}]}`},
			},
			wantText: "Snapshot policy created successfully",
		},
		{
			name: "create policy every5min with existing schedule",
			tool: "create_snapshot_policy",
			args: policyArgs("every5min", "5minutes", 2),
			ontap: []exchange{
				countSchedules("5minutes", `{"num_records":1}`),
				{method: http.MethodPost, path: policies, status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"every5min","copies":[{"count":2,"schedule":{"name":"5minutes"}}]}`},
			},
			wantText: "Snapshot policy created successfully",
		},
		{
			// The prompts say "if not exist then create schedule first"; the agent learns that from this message.
			name:     "create policy with missing schedule stops before POST",
			tool:     "create_snapshot_policy",
			args:     policyArgs("every4hours", "4hours", 5),
			ontap:    []exchange{countSchedules("4hours", `{"num_records":0}`)},
			wantText: "no schedule 4hours found",
			wantErr:  true,
		},
		{
			name: "create policy surfaces ONTAP error message",
			tool: "create_snapshot_policy",
			args: policyArgs("every4hours", "4hours", 5),
			ontap: []exchange{
				countSchedules("4hours", `{"num_records":1}`),
				{method: http.MethodPost, path: policies, status: http.StatusConflict,
					body: `{"svm":{"name":"marketing"},"name":"every4hours","copies":[{"count":5,"schedule":{"name":"4hours"}}]}`,
					resp: `{"error":{"message":"Snapshot policy \"every4hours\" already exists.","code":"1638413"}}`},
			},
			wantText: `message: Snapshot policy "every4hours" already exists. code: 1638413`,
			wantErr:  true,
		},
		{
			name:     "create policy without count sends nothing",
			tool:     "create_snapshot_policy",
			args:     policyArgs("every4hours", "4hours", 0),
			wantText: "snapshot copies count is required",
			wantErr:  true,
		},

		// add_schedule_in_snapshot_policy
		{
			name: "add schedule 2hours keeping 6",
			tool: "add_schedule_in_snapshot_policy",
			args: addEntryArgs,
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				{method: http.MethodPost, path: entries, status: http.StatusCreated, body: `{"count":6,"schedule":{"name":"2hours"}}`},
			},
			wantText: "Schedule added to snapshot policy successfully",
		},
		{
			name:     "add schedule to missing policy",
			tool:     "add_schedule_in_snapshot_policy",
			args:     addEntryArgs,
			ontap:    []exchange{lookupPolicy("every4hours", empty)},
			wantText: policyGone,
			wantErr:  true,
		},
		{
			name: "add schedule surfaces ONTAP error message",
			tool: "add_schedule_in_snapshot_policy",
			args: addEntryArgs,
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				{method: http.MethodPost, path: entries, status: http.StatusBadRequest, body: `{"count":6,"schedule":{"name":"2hours"}}`,
					resp: `{"error":{"message":"Schedule \"2hours\" does not exist.","code":"1638505"}}`},
			},
			wantText: `message: Schedule "2hours" does not exist. code: 1638505`,
			wantErr:  true,
		},

		// modify_schedule_in_snapshot_policy
		{
			name: "update schedule entry snapmirror label only",
			tool: "modify_schedule_in_snapshot_policy",
			args: entryArgs("update", map[string]any{"snapmirror_label": "sm2"}),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				lookupEntry(entryUUID),
				{method: http.MethodPatch, path: entryAt, body: `{"snapmirror_label":"sm2"}`},
			},
			wantText: "Schedule updated in snapshot policy successfully",
		},
		{
			name: "update schedule entry missing from policy",
			tool: "modify_schedule_in_snapshot_policy",
			args: entryArgs("update", map[string]any{"snapmirror_label": "sm2"}),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				lookupEntry(empty),
			},
			wantText: entryAbsent,
			wantErr:  true,
		},
		{
			name:     "update schedule entry in missing policy",
			tool:     "modify_schedule_in_snapshot_policy",
			args:     entryArgs("update", map[string]any{"count": 3}),
			ontap:    []exchange{lookupPolicy("every4hours", empty)},
			wantText: policyGone,
			wantErr:  true,
		},
		{
			name:     "update schedule entry with no fields sends nothing",
			tool:     "modify_schedule_in_snapshot_policy",
			args:     entryArgs("update", nil),
			wantText: "at least one updatable field must be provided: count or snapmirror_label",
			wantErr:  true,
		},
		{
			name: "update schedule entry surfaces ONTAP error message",
			tool: "modify_schedule_in_snapshot_policy",
			args: entryArgs("update", map[string]any{"count": 300}),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				lookupEntry(entryUUID),
				{method: http.MethodPatch, path: entryAt, body: `{"count":300}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Total count exceeds 1023.","code":"1638507"}}`},
			},
			wantText: "message: Total count exceeds 1023. code: 1638507",
			wantErr:  true,
		},
		{
			name: "remove schedule entry",
			tool: "modify_schedule_in_snapshot_policy",
			args: entryArgs("remove", nil),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				lookupEntry(entryUUID),
				{method: http.MethodDelete, path: entryAt},
			},
			wantText: "Schedule removed from snapshot policy successfully",
		},
		{
			name: "remove schedule entry missing from policy",
			tool: "modify_schedule_in_snapshot_policy",
			args: entryArgs("remove", nil),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				lookupEntry(empty),
			},
			wantText: entryAbsent,
			wantErr:  true,
		},
		{
			name:     "modify schedule entry rejects unknown operation",
			tool:     "modify_schedule_in_snapshot_policy",
			args:     entryArgs("add", nil),
			wantText: `unsupported operation "add"; supported values: update, remove`,
			wantErr:  true,
		},

		// modify_snapshot_policy
		{
			name: "disable policy with comment",
			tool: "modify_snapshot_policy",
			args: modifyPolicyArgs("update", map[string]any{"enabled": "false", "comment": "4_hour_policy"}),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				{method: http.MethodPatch, path: policyAt, body: `{"enabled":"false","comment":"4_hour_policy"}`},
			},
			wantText: "Snapshot policy updated successfully",
		},
		{
			name:     "update missing policy",
			tool:     "modify_snapshot_policy",
			args:     modifyPolicyArgs("update", map[string]any{"comment": "4_hour_policy"}),
			ontap:    []exchange{lookupPolicy("every4hours", empty)},
			wantText: policyGone,
			wantErr:  true,
		},
		{
			name:     "update policy with no fields sends nothing",
			tool:     "modify_snapshot_policy",
			args:     modifyPolicyArgs("update", nil),
			wantText: "at least one updatable field must be provided: comment or enabled",
			wantErr:  true,
		},
		{
			name: "delete policy by UUID",
			tool: "modify_snapshot_policy",
			args: modifyPolicyArgs("delete", nil),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				{method: http.MethodDelete, path: policyAt},
			},
			wantText: "Snapshot policy deleted successfully",
		},
		{
			// The integration "Clean snapshot policy" steps run before the policies exist.
			name:     "delete of missing policy",
			tool:     "modify_snapshot_policy",
			args:     modifyPolicyArgs("delete", nil),
			ontap:    []exchange{lookupPolicy("every4hours", empty)},
			wantText: policyGone,
			wantErr:  true,
		},
		{
			name: "delete policy surfaces ONTAP error message",
			tool: "modify_snapshot_policy",
			args: modifyPolicyArgs("delete", nil),
			ontap: []exchange{
				lookupPolicy("every4hours", policyUUID),
				{method: http.MethodDelete, path: policyAt, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Snapshot policy is in use by volume docs.","code":"1638419"}}`},
			},
			wantText: "message: Snapshot policy is in use by volume docs. code: 1638419",
			wantErr:  true,
		},
		{
			name:     "modify policy rejects unknown operation",
			tool:     "modify_snapshot_policy",
			args:     modifyPolicyArgs("disable", nil),
			wantText: `unsupported operation "disable"; supported values: update, delete`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_snapshot_policy",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "marketing", "name": "every4hours"},
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
