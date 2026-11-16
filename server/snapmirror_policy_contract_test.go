package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

const (
	snapMirrorPoliciesPath     = "/api/snapmirror/policies"
	snapMirrorPolicyBaseFields = "uuid,name,type,comment,retention,network_compression_enabled,throttle"
	snapMirrorPolicyFields     = snapMirrorPolicyBaseFields + ",create_snapshot_on_source,copy_latest_source_snapshot"
	snapMirrorPolicyUUID       = "policy-uuid"
	snapMirrorPolicyName       = "daily-retention"
	snapMirrorPolicyState      = `{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","comment":"custom retention","network_compression_enabled":true,"throttle":50000,"retention":[{"label":"daily","count":7},{"label":"weekly","count":4,"preserve":true,"warn":2}]}]}`
)

func snapMirrorPolicyVersion(version string) exchange {
	generation, major := "9", version
	if before, after, ok := strings.Cut(version, "."); ok {
		generation, major = before, after
	}
	return exchange{method: http.MethodGet, path: "/api/cluster?fields=*",
		resp: `{"version":{"generation":` + generation + `,"major":` + major + `}}`}
}

func snapMirrorPolicyLookup(resp string) exchange {
	return snapMirrorPolicyLookupFields(resp, snapMirrorPolicyFields)
}

func snapMirrorPolicyLookupFields(resp, fields string) exchange {
	return exchange{method: http.MethodGet,
		path: "/api/snapmirror/policies?fields=" + fields + "&name=" + snapMirrorPolicyName + "&scope=cluster",
		resp: resp}
}

func TestSnapMirrorPolicyContract(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText []string
		wantErr  bool
	}{
		{
			name: "create policy returns rules and warns for unmatched schedule label",
			tool: "create_snapmirror_policy",
			args: map[string]any{
				"cluster_name": contractCluster, "name": snapMirrorPolicyName, "policy_type": "mirror_vault",
				"comment": "custom retention", "network_compression_enabled": true, "throttle": 50000,
				"retention_rules": []any{
					map[string]any{"label": "daily", "keep": 7},
					map[string]any{"label": "weekly", "keep": 4, "preserve": true, "warn": 2},
				},
			},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				{method: http.MethodPost, path: snapMirrorPoliciesPath, status: http.StatusCreated,
					body: `{"name":"daily-retention","type":"async","comment":"custom retention","network_compression_enabled":true,"throttle":50000,"retention":[{"label":"daily","count":7},{"label":"weekly","count":4,"preserve":true,"warn":2}]}`},
				snapMirrorPolicyLookup(snapMirrorPolicyState),
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=daily&fields=uuid&return_records=false", resp: `{"num_records":1}`},
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=weekly&fields=uuid&return_records=false", resp: `{"num_records":0}`},
			},
			wantText: []string{
				`SnapMirror policy "daily-retention" created successfully`,
				`label="daily", keep=7; snapshot schedule label: matched`,
				`label="weekly", keep=4, preserve=true, warn=2; WARNING: no snapshot policy schedule entry produces this snapmirror_label`,
				`accepts transfer attributes: network_compression_enabled, throttle`,
			},
		},
		{
			name: "older supported ONTAP version uses fields available in its release",
			tool: "create_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "name": snapMirrorPolicyName, "policy_type": "async"},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.10"),
				{method: http.MethodPost, path: snapMirrorPoliciesPath, status: http.StatusCreated,
					body: `{"name":"daily-retention","type":"async"}`},
				snapMirrorPolicyLookupFields(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async"}]}`, snapMirrorPolicyBaseFields),
			},
			wantText: []string{`SnapMirror policy "daily-retention" created successfully`, `Target cluster "contract-cluster" accepts transfer attributes: network_compression_enabled, throttle.`},
		},
		{
			name: "update policy properties",
			tool: "update_snapmirror_policy",
			args: map[string]any{
				"cluster_name": contractCluster, "policy_name": snapMirrorPolicyName,
				"snapmirror_policy_update": map[string]any{"comment": "revised", "throttle": 60000},
			},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				snapMirrorPolicyLookup(snapMirrorPolicyState),
				{method: http.MethodPatch, path: snapMirrorPoliciesPath + "/" + snapMirrorPolicyUUID,
					body: `{"comment":"revised","throttle":60000}`},
				snapMirrorPolicyLookup(strings.Replace(strings.Replace(snapMirrorPolicyState, `"comment":"custom retention"`, `"comment":"revised"`, 1), `"throttle":50000`, `"throttle":60000`, 1)),
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=daily&fields=uuid&return_records=false", resp: `{"num_records":1}`},
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=weekly&fields=uuid&return_records=false", resp: `{"num_records":0}`},
			},
			wantText: []string{`SnapMirror policy "daily-retention" updated successfully`, `throttle=60000 KB/s`, `WARNING: no snapshot policy schedule entry`},
		},
		{
			name: "add retention rule",
			tool: "add_rule_in_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "policy_name": snapMirrorPolicyName, "rule_label": "monthly", "keep_count": 3},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[{"label":"daily","count":7}]}]}`),
				{method: http.MethodPatch, path: snapMirrorPoliciesPath + "/" + snapMirrorPolicyUUID,
					body: `{"retention":[{"label":"daily","count":7},{"label":"monthly","count":3}]}`},
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[{"label":"daily","count":7},{"label":"monthly","count":3}]}]}`),
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=daily&fields=uuid&return_records=false", resp: `{"num_records":1}`},
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=monthly&fields=uuid&return_records=false", resp: `{"num_records":1}`},
			},
			wantText: []string{`Retention rule added to SnapMirror policy "daily-retention" successfully`, `label="monthly", keep=3; snapshot schedule label: matched`},
		},
		{
			name: "update retention rule count",
			tool: "update_rule_in_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "policy_name": snapMirrorPolicyName, "rule_label": "daily", "keep_count": 14},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[{"label":"daily","count":7}]}]}`),
				{method: http.MethodPatch, path: snapMirrorPoliciesPath + "/" + snapMirrorPolicyUUID,
					body: `{"retention":[{"label":"daily","count":14}]}`},
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[{"label":"daily","count":14}]}]}`),
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=daily&fields=uuid&return_records=false", resp: `{"num_records":1}`},
			},
			wantText: []string{`Retention rule updated in SnapMirror policy "daily-retention" successfully`, `label="daily", keep=14; snapshot schedule label: matched`},
		},
		{
			name: "update preserve without changing keep count",
			tool: "update_rule_in_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "policy_name": snapMirrorPolicyName, "rule_label": "daily", "preserve": true},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[{"label":"daily","count":7}]}]}`),
				{method: http.MethodPatch, path: snapMirrorPoliciesPath + "/" + snapMirrorPolicyUUID,
					body: `{"retention":[{"label":"daily","count":7,"preserve":true}]}`},
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[{"label":"daily","count":7,"preserve":true}]}]}`),
				{method: http.MethodGet, path: "/api/storage/snapshot-policies?copies.snapmirror_label=daily&fields=uuid&return_records=false", resp: `{"num_records":1}`},
			},
			wantText: []string{`Retention rule updated in SnapMirror policy "daily-retention" successfully`, `label="daily", keep=7, preserve=true; snapshot schedule label: matched`},
		},
		{
			name: "remove retention rule",
			tool: "remove_rule_in_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "policy_name": snapMirrorPolicyName, "rule_label": "daily"},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[{"label":"daily","count":7}]}]}`),
				{method: http.MethodPatch, path: snapMirrorPoliciesPath + "/" + snapMirrorPolicyUUID, body: `{"retention":[]}`},
				snapMirrorPolicyLookup(`{"num_records":1,"records":[{"uuid":"policy-uuid","name":"daily-retention","type":"async","retention":[]}]}`),
			},
			wantText: []string{`Retention rule removed from SnapMirror policy "daily-retention" successfully`, "Retention rules:\n- none"},
		},
		{
			name: "delete policy names blocking relationships",
			tool: "delete_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "name": snapMirrorPolicyName},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				snapMirrorPolicyLookup(snapMirrorPolicyState),
				{method: http.MethodGet, path: "/api/snapmirror/relationships?fields=source.path,destination.path&max_records=10000&policy.name=daily-retention",
					resp: `{"num_records":1,"records":[{"source":{"path":"src:vol"},"destination":{"path":"dst:vol"}}]}`},
			},
			wantText: []string{`cannot delete SnapMirror policy "daily-retention" because it is in use`, "src:vol -> dst:vol", "reassign or delete those relationships first"},
			wantErr:  true,
		},
		{
			name:     "built-in policy cannot be modified",
			tool:     "update_snapmirror_policy",
			args:     map[string]any{"cluster_name": contractCluster, "policy_name": "MirrorAndVault", "snapmirror_policy_update": map[string]any{"throttle": 100}},
			wantText: []string{`SnapMirror policy "MirrorAndVault" is built-in and cannot be modified`, "create a custom policy instead"},
			wantErr:  true,
		},
		{
			name: "preserve setting rejected on older ONTAP",
			tool: "add_rule_in_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "policy_name": snapMirrorPolicyName, "rule_label": "daily", "keep_count": 2, "preserve": true},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.12"),
			},
			wantText: []string{"ONTAP 9.12 does not support SnapMirror retention rule preserve or warn settings"},
			wantErr:  true,
		},
		{
			name: "create preserves ONTAP error code and message",
			tool: "create_snapmirror_policy",
			args: map[string]any{"cluster_name": contractCluster, "name": snapMirrorPolicyName, "policy_type": "mirror_vault"},
			ontap: []exchange{
				snapMirrorPolicyVersion("9.15"),
				{method: http.MethodPost, path: snapMirrorPoliciesPath, body: `{"name":"daily-retention","type":"async"}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"invalid retention policy","code":"12345"}}`},
			},
			wantText: []string{"message: invalid retention policy code: 12345"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "both", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			if isErr != tt.wantErr {
				t.Fatalf("callTool() error = %v, want %v; text: %s", isErr, tt.wantErr, text)
			}
			for _, want := range tt.wantText {
				if !strings.Contains(text, want) {
					t.Errorf("callTool() text missing %q:\n%s", want, text)
				}
			}
		})
	}
}

func TestNewSnapMirrorPolicyTypeMapping(t *testing.T) {
	remote := ontap.Remote{Version: ontap.Version{Generation: 9, Major: 15}}
	tests := []struct {
		name               string
		policyType         string
		retentionRules     []tool.SnapMirrorPolicyRule
		wantCreateSnapshot *bool
		wantCopyLatest     *bool
		wantErr            bool
	}{
		{name: "async", policyType: "async"},
		{name: "vault", policyType: "vault", retentionRules: []tool.SnapMirrorPolicyRule{{Label: "daily", Keep: 7}}, wantCreateSnapshot: new(false)},
		{name: "mirror vault", policyType: "mirror_vault", retentionRules: []tool.SnapMirrorPolicyRule{{Label: "daily", Keep: 7}}},
		{name: "mirror-vault alias", policyType: "mirror-vault", retentionRules: []tool.SnapMirrorPolicyRule{{Label: "daily", Keep: 7}}},
		{name: "async mirror", policyType: "async_mirror", wantCopyLatest: new(true)},
		{name: "async mirror rejects custom retention", policyType: "async_mirror", retentionRules: []tool.SnapMirrorPolicyRule{{Label: "daily", Keep: 7}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := newSnapMirrorPolicy(tool.SnapMirrorPolicyCreate{
				Name: "custom-policy", Type: tt.policyType, RetentionRules: tt.retentionRules,
			}, remote)
			if (err != nil) != tt.wantErr {
				t.Fatalf("newSnapMirrorPolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if policy.Type != "async" {
				t.Errorf("policy type = %q, want async REST type", policy.Type)
			}
			if !sameBoolPointer(policy.CreateSnapshotOnSource, tt.wantCreateSnapshot) {
				t.Errorf("create_snapshot_on_source = %v, want %v", policy.CreateSnapshotOnSource, tt.wantCreateSnapshot)
			}
			if !sameBoolPointer(policy.CopyLatestSourceSnapshot, tt.wantCopyLatest) {
				t.Errorf("copy_latest_source_snapshot = %v, want %v", policy.CopyLatestSourceSnapshot, tt.wantCopyLatest)
			}
		})
	}
}

func sameBoolPointer(got, want *bool) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}

func TestSnapMirrorPolicyDeleteSucceedsWhenUnused(t *testing.T) {
	script := []exchange{
		snapMirrorPolicyVersion("9.15"),
		snapMirrorPolicyLookup(snapMirrorPolicyState),
		{method: http.MethodGet, path: "/api/snapmirror/relationships?fields=source.path,destination.path&max_records=10000&policy.name=daily-retention", resp: `{"num_records":0,"records":[]}`},
		{method: http.MethodDelete, path: snapMirrorPoliciesPath + "/" + snapMirrorPolicyUUID},
	}
	cs := newContractSession(t, "both", script)
	text, isErr := callTool(t, cs, "delete_snapmirror_policy", map[string]any{"cluster_name": contractCluster, "name": snapMirrorPolicyName})
	if isErr || !strings.Contains(text, `SnapMirror policy "daily-retention" deleted successfully`) {
		t.Fatalf("callTool() = (%q, %v), want successful delete", text, isErr)
	}
}
