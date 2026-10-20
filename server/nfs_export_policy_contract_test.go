package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// nfsLookupPolicy is the SVM-scoped name-to-id lookup that precedes every export policy update,
// delete, and rule operation. It returns one record per id; no ids means the policy does not exist.
func nfsLookupPolicy(svm, name string, ids ...int) exchange {
	records := make([]string, 0, len(ids))
	for _, id := range ids {
		records = append(records, `{"id":`+strconv.Itoa(id)+`}`)
	}
	return exchange{
		method: http.MethodGet,
		path:   "/api/protocols/nfs/export-policies?fields=id&name=" + name + "&svm.name=" + svm,
		resp:   `{"num_records":` + strconv.Itoa(len(ids)) + `,"records":[` + strings.Join(records, ",") + `]}`,
	}
}

func TestNFSExportPolicyContract(t *testing.T) {
	const (
		policiesPath  = "/api/protocols/nfs/export-policies"
		policyPath    = policiesPath + "/42"
		rulesPath     = policyPath + "/rules"
		ruleFound     = `{"num_records":1,"records":[{"index":3}]}`
		ruleMissing   = `{"num_records":0,"records":[]}`
		ruleAmbiguous = `{"num_records":2,"records":[{"index":3},{"index":4}]}`
	)

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		// create_nfs_export_policies. The input schema marks client_match, ro_rule, and rw_rule as
		// required although the handler treats them as optional, so policy-only creates pass them empty.
		{
			name: "create policy",
			tool: "create_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client_match": "", "ro_rule": "", "rw_rule": ""},
			ontap: []exchange{
				{method: http.MethodPost, path: policiesPath, status: http.StatusCreated, body: `{"svm":{"name":"marketing"},"name":"nfsEngPolicy"}`},
			},
			wantText: "NFS Export Policy created successfully",
		},
		{
			name: "create policy with an initial rule splits rule lists",
			tool: "create_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy",
				"client_match": "0.0.0.0/0", "ro_rule": "any", "rw_rule": "sys,krb5"},
			ontap: []exchange{
				{method: http.MethodPost, path: policiesPath, status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"nfsEngPolicy","rules":[{"clients":[{"match":"0.0.0.0/0"}],"ro_rule":["any"],"rw_rule":["sys","krb5"]}]}`},
			},
			wantText: "NFS Export Policy created successfully",
		},
		{
			// The initial rule fields are optional; the schema used to mark them required.
			name: "create policy without rule fields",
			tool: "create_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy"},
			ontap: []exchange{
				{method: http.MethodPost, path: policiesPath, status: http.StatusCreated, body: `{"svm":{"name":"marketing"},"name":"nfsEngPolicy"}`},
			},
			wantText: "NFS Export Policy created successfully",
		},
		{
			name: "create policy trims spaces in rule lists",
			tool: "create_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy",
				"client_match": "0.0.0.0/0", "ro_rule": " any ", "rw_rule": "sys, krb5,"},
			ontap: []exchange{
				{method: http.MethodPost, path: policiesPath, status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"nfsEngPolicy","rules":[{"clients":[{"match":"0.0.0.0/0"}],"ro_rule":["any"],"rw_rule":["sys","krb5"]}]}`},
			},
			wantText: "NFS Export Policy created successfully",
		},
		{
			name:     "create with a partial rule is rejected without calling ONTAP",
			tool:     "create_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client_match": "0.0.0.0/0", "ro_rule": "", "rw_rule": ""},
			wantText: "read only rules are required",
			wantErr:  true,
		},
		{
			name:     "create without SVM is rejected without calling ONTAP",
			tool:     "create_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "", "export_policy": "nfsEngPolicy", "client_match": "", "ro_rule": "", "rw_rule": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name: "create surfaces ONTAP error message",
			tool: "create_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client_match": "", "ro_rule": "", "rw_rule": ""},
			ontap: []exchange{
				{method: http.MethodPost, path: policiesPath, status: http.StatusConflict, body: `{"svm":{"name":"marketing"},"name":"nfsEngPolicy"}`,
					resp: `{"error":{"message":"Duplicate export policy name nfsEngPolicy.","code":"1703936"}}`},
			},
			wantText: "Duplicate export policy name nfsEngPolicy.",
			wantErr:  true,
		},

		// modify_nfs_export_policies
		{
			name: "rename policy by id",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_update": map[string]any{"new_export_policy": "nfsMgrPolicy"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodPatch, path: policyPath, body: `{"name":"nfsMgrPolicy"}`},
			},
			wantText: "NFS Export Policy updated successfully",
		},
		{
			name: "update replaces rules",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_update": map[string]any{"client_match": "10.0.0.0/24", "ro_rule": "sys", "rw_rule": "none"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodPatch, path: policyPath, body: `{"rules":[{"clients":[{"match":"10.0.0.0/24"}],"ro_rule":["sys"],"rw_rule":["none"]}]}`},
			},
			wantText: "NFS Export Policy updated successfully",
		},
		{
			name: "update of missing policy",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "ghost",
				"nfs_export_policy_update": map[string]any{"new_export_policy": "nfsMgrPolicy"}},
			ontap:    []exchange{nfsLookupPolicy("marketing", "ghost")},
			wantText: "failed to get detail of export policy ghost because it does not exist",
			wantErr:  true,
		},
		{
			name:     "update with no fields is rejected without calling ONTAP",
			tool:     "modify_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy"},
			wantText: "at least one updatable field must be provided",
			wantErr:  true,
		},
		{
			name: "update with a partial rule is rejected without calling ONTAP",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_update": map[string]any{"client_match": "10.0.0.0/24", "ro_rule": "sys"}},
			wantText: "rw_rule is required",
			wantErr:  true,
		},
		{
			name: "update surfaces ONTAP error message",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_update": map[string]any{"new_export_policy": "nfsMgrPolicy"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodPatch, path: policyPath, body: `{"name":"nfsMgrPolicy"}`, status: http.StatusConflict,
					resp: `{"error":{"message":"Duplicate export policy name nfsMgrPolicy.","code":"1703936"}}`},
			},
			wantText: "Duplicate export policy name nfsMgrPolicy.",
			wantErr:  true,
		},
		{
			name: "delete policy by id",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "nfsMgrPolicy"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsMgrPolicy", 42),
				{method: http.MethodDelete, path: policyPath},
			},
			wantText: "NFS Export Policy deleted successfully",
		},
		{
			// The integration suite's "Clean NFS export policy" steps depend on this message.
			name:     "delete of missing policy",
			tool:     "modify_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "ghost"},
			ontap:    []exchange{nfsLookupPolicy("marketing", "ghost")},
			wantText: "failed to get detail of export policy ghost because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete surfaces ONTAP error message",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "nfsMgrPolicy"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsMgrPolicy", 42),
				{method: http.MethodDelete, path: policyPath, status: http.StatusConflict,
					resp: `{"error":{"message":"Cannot delete export policy nfsMgrPolicy because it is in use by volume docs.","code":"1703954"}}`},
			},
			wantText: "because it is in use by volume docs.",
			wantErr:  true,
		},
		{
			// Every SVM has a policy named "default", so the lookup must be scoped to the requested SVM.
			name: "delete of default policy is scoped to the SVM",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "finance", "operation": "delete", "export_policy": "default"},
			ontap: []exchange{
				nfsLookupPolicy("finance", "default", 22),
				{method: http.MethodDelete, path: policiesPath + "/22"},
			},
			wantText: "NFS Export Policy deleted successfully",
		},
		{
			// If the lookup ever matches more than one policy, the tool must refuse rather than act on the first record.
			name:     "delete of ambiguous policy is refused",
			tool:     "modify_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "default"},
			ontap:    []exchange{nfsLookupPolicy("marketing", "default", 11, 22)},
			wantText: "found 2 export policies named default on svm marketing, expected 1",
			wantErr:  true,
		},
		{
			name: "update of ambiguous policy is refused",
			tool: "modify_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "default",
				"nfs_export_policy_update": map[string]any{"new_export_policy": "nfsMgrPolicy"}},
			ontap:    []exchange{nfsLookupPolicy("marketing", "default", 11, 22)},
			wantText: "found 2 export policies named default on svm marketing, expected 1",
			wantErr:  true,
		},
		{
			name:     "modify without SVM is rejected without calling ONTAP",
			tool:     "modify_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "", "operation": "delete", "export_policy": "default"},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name:     "modify without policy name is rejected without calling ONTAP",
			tool:     "modify_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": ""},
			wantText: "export policy name is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "rename", "export_policy": "nfsEngPolicy"},
			wantText: `unsupported operation "rename"`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "modify_nfs_export_policies",
			args:     map[string]any{"cluster_name": "nope", "svm_name": "marketing", "operation": "delete", "export_policy": "nfsEngPolicy"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},

		// Attach the policy to a volume, as the integration test does through modify_volume.
		{
			name: "attach policy to volume",
			tool: "modify_volume",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "volume_name": "docs",
				"volume_update": map[string]any{"nas.export_policy.name": "nfsEngPolicy"}},
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/storage/volumes?fields=uuid&svm.name=marketing&name=docs", resp: `{"num_records":1,"records":[{"uuid":"vol-uuid"}]}`},
				{method: http.MethodPatch, path: "/api/storage/volumes/vol-uuid", body: `{"nas":{"export_policy":{"name":"nfsEngPolicy"}}}`,
					status: http.StatusAccepted, resp: `{"job":{"uuid":"nfs-job"}}`},
				{method: http.MethodGet, path: "/api/cluster/jobs/nfs-job", resp: `{"uuid":"nfs-job","state":"success"}`},
			},
			wantText: "Volume updated successfully",
		},

		// create_nfs_export_policies_rules
		{
			name: "add rule to policy",
			tool: "create_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client": "10.0.0.0/24", "ro_rule": "sys", "rw_rule": "sys,ntlm"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodPost, path: rulesPath, status: http.StatusCreated,
					body: `{"clients":[{"match":"10.0.0.0/24"}],"ro_rule":["sys"],"rw_rule":["sys","ntlm"]}`},
			},
			wantText: "NFS Export Policy Rules created successfully",
		},
		{
			name: "add rule trims spaces in rule lists",
			tool: "create_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client": "10.0.0.0/24", "ro_rule": "sys", "rw_rule": "sys , ntlm"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodPost, path: rulesPath, status: http.StatusCreated,
					body: `{"clients":[{"match":"10.0.0.0/24"}],"ro_rule":["sys"],"rw_rule":["sys","ntlm"]}`},
			},
			wantText: "NFS Export Policy Rules created successfully",
		},
		{
			name:     "add rule with only separators in a rule list is rejected without calling ONTAP",
			tool:     "create_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client": "10.0.0.0/24", "ro_rule": "sys", "rw_rule": " , "},
			wantText: "read write rules are required",
			wantErr:  true,
		},
		{
			name:     "add rule to missing policy",
			tool:     "create_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "ghost", "client": "10.0.0.0/24", "ro_rule": "sys", "rw_rule": "sys"},
			ontap:    []exchange{nfsLookupPolicy("marketing", "ghost")},
			wantText: "failed to get detail of export policy ghost because it does not exist",
			wantErr:  true,
		},
		{
			name:     "add rule without client is rejected without calling ONTAP",
			tool:     "create_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client": "", "ro_rule": "sys", "rw_rule": "sys"},
			wantText: "client match is required",
			wantErr:  true,
		},
		{
			name: "add rule surfaces ONTAP error message",
			tool: "create_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "client": "10.0.0.0/24", "ro_rule": "bogus", "rw_rule": "sys"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodPost, path: rulesPath, status: http.StatusBadRequest,
					body: `{"clients":[{"match":"10.0.0.0/24"}],"ro_rule":["bogus"],"rw_rule":["sys"]}`,
					resp: `{"error":{"message":"bogus is an invalid value for field ro_rule","code":"262196"}}`},
			},
			wantText: "bogus is an invalid value for field ro_rule",
			wantErr:  true,
		},

		{
			name:     "add rule to ambiguous policy is refused",
			tool:     "create_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "default", "client": "10.0.0.0/24", "ro_rule": "sys", "rw_rule": "sys"},
			ontap:    []exchange{nfsLookupPolicy("marketing", "default", 11, 22)},
			wantText: "found 2 export policies named default on svm marketing, expected 1",
			wantErr:  true,
		},
		{
			name:     "add rule without SVM is rejected without calling ONTAP",
			tool:     "create_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "", "export_policy": "nfsEngPolicy", "client": "10.0.0.0/24", "ro_rule": "sys", "rw_rule": "sys"},
			wantText: "SVM name is required",
			wantErr:  true,
		},

		// modify_nfs_export_policies_rules
		{
			name: "update rule found by old client",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "10.0.0.0/24", "client": "10.0.1.0/24", "rw_rule": "none"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=10.0.0.0/24", resp: ruleFound},
				{method: http.MethodPatch, path: rulesPath + "/3", body: `{"clients":[{"match":"10.0.1.0/24"}],"rw_rule":["none"]}`},
			},
			wantText: "NFS Export Policy Rule updated successfully",
		},
		{
			name: "update rule found by old ro and rw rules",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_ro_rule": "sys", "old_rw_rule": "sys", "ro_rule": "krb5,krb5i"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&ro_rule=sys&rw_rule=sys", resp: ruleFound},
				{method: http.MethodPatch, path: rulesPath + "/3", body: `{"ro_rule":["krb5","krb5i"]}`},
			},
			wantText: "NFS Export Policy Rule updated successfully",
		},
		{
			name: "update of missing rule",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "10.9.9.9", "client": "10.0.1.0/24"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=10.9.9.9", resp: ruleMissing},
			},
			wantText: "failed to get detail of export policy rule because it does not exist",
			wantErr:  true,
		},
		{
			name: "update rule in missing policy",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "ghost",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "10.0.0.0/24", "client": "10.0.1.0/24"}},
			ontap:    []exchange{nfsLookupPolicy("marketing", "ghost")},
			wantText: "failed to get detail of export policy ghost because it does not exist",
			wantErr:  true,
		},
		{
			name: "update rule without old selector is rejected without calling ONTAP",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"client": "10.0.1.0/24"}},
			wantText: "old client match OR old ro rule OR old rw rules are required",
			wantErr:  true,
		},
		{
			name: "update rule with no new fields is rejected without calling ONTAP",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "10.0.0.0/24"}},
			wantText: "at least one updatable field must be provided",
			wantErr:  true,
		},
		{
			name: "update rule surfaces ONTAP error message",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "10.0.0.0/24", "ro_rule": "bogus"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=10.0.0.0/24", resp: ruleFound},
				{method: http.MethodPatch, path: rulesPath + "/3", body: `{"ro_rule":["bogus"]}`, status: http.StatusBadRequest,
					resp: `{"error":{"message":"invalid ro_rule value bogus","code":"262196"}}`},
			},
			wantText: "invalid ro_rule value bogus",
			wantErr:  true,
		},
		{
			name: "delete rule found by old client and ro rule",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "10.0.0.0/24", "old_ro_rule": "sys"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=10.0.0.0/24&ro_rule=sys", resp: ruleFound},
				{method: http.MethodDelete, path: rulesPath + "/3"},
			},
			wantText: "NFS Export Policy Rule deleted successfully",
		},
		{
			name: "delete of missing rule",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "10.9.9.9"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=10.9.9.9", resp: ruleMissing},
			},
			wantText: "failed to get detail of export policy rule because it does not exist",
			wantErr:  true,
		},
		{
			name:     "delete rule without old selector is rejected without calling ONTAP",
			tool:     "modify_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "nfsEngPolicy"},
			wantText: "old client match OR old ro rule OR old rw rules are required",
			wantErr:  true,
		},
		{
			name: "update of ambiguous rule is refused",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "update", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_ro_rule": "sys", "rw_rule": "none"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&ro_rule=sys", resp: ruleAmbiguous},
			},
			wantText: `found 2 export policy rules matching old_client="" old_ro_rule="sys" old_rw_rule="" in export policy nfsEngPolicy on svm marketing, expected 1`,
			wantErr:  true,
		},
		{
			name: "delete of ambiguous rule is refused",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "0.0.0.0/0"}},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=0.0.0.0/0", resp: ruleAmbiguous},
			},
			wantText: `found 2 export policy rules matching old_client="0.0.0.0/0" old_ro_rule="" old_rw_rule="" in export policy nfsEngPolicy on svm marketing, expected 1`,
			wantErr:  true,
		},
		{
			name: "delete rule in ambiguous policy is refused",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "delete", "export_policy": "default",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "0.0.0.0/0"}},
			ontap:    []exchange{nfsLookupPolicy("marketing", "default", 11, 22)},
			wantText: "found 2 export policies named default on svm marketing, expected 1",
			wantErr:  true,
		},
		{
			name: "modify rules without SVM is rejected without calling ONTAP",
			tool: "modify_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "", "operation": "delete", "export_policy": "nfsEngPolicy",
				"nfs_export_policy_rules_update": map[string]any{"old_client": "0.0.0.0/0"}},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name:     "modify rules rejects unknown operation",
			tool:     "modify_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "operation": "add", "export_policy": "nfsEngPolicy"},
			wantText: `unsupported operation "add"`,
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

// The legacy update/delete tools share the REST lookups, so they must scope by SVM too.
func TestNFSExportPolicyLegacyContract(t *testing.T) {
	const rulesPath = "/api/protocols/nfs/export-policies/42/rules"

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			name: "update policy",
			tool: "update_nfs_export_policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "new_export_policy": "nfsMgrPolicy"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodPatch, path: "/api/protocols/nfs/export-policies/42", body: `{"name":"nfsMgrPolicy"}`},
			},
			wantText: "NFS Export Policy updated successfully",
		},
		{
			name:     "delete ambiguous policy is refused",
			tool:     "delete_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "default"},
			ontap:    []exchange{nfsLookupPolicy("marketing", "default", 11, 22)},
			wantText: "found 2 export policies named default on svm marketing, expected 1",
			wantErr:  true,
		},
		{
			name:     "delete policy without SVM is rejected without calling ONTAP",
			tool:     "delete_nfs_export_policies",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "", "export_policy": "default"},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name: "update rule",
			tool: "update_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "old_client": "10.0.0.0/24", "rw_rule": "none"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=10.0.0.0/24", resp: `{"num_records":1,"records":[{"index":3}]}`},
				{method: http.MethodPatch, path: rulesPath + "/3", body: `{"rw_rule":["none"]}`},
			},
			wantText: "NFS Export Policy Rules updated successfully",
		},
		{
			name: "delete ambiguous rule is refused",
			tool: "delete_nfs_export_policies_rules",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "export_policy": "nfsEngPolicy", "old_client": "0.0.0.0/0"},
			ontap: []exchange{
				nfsLookupPolicy("marketing", "nfsEngPolicy", 42),
				{method: http.MethodGet, path: rulesPath + "?fields=index&clients.match=0.0.0.0/0", resp: `{"num_records":2,"records":[{"index":3},{"index":4}]}`},
			},
			wantText: "found 2 export policy rules matching",
			wantErr:  true,
		},
		{
			name:     "delete rule without SVM is rejected without calling ONTAP",
			tool:     "delete_nfs_export_policies_rules",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "", "export_policy": "nfsEngPolicy", "old_client": "0.0.0.0/0"},
			wantText: "SVM name is required",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "legacy", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("got: %q; want to contain: %q", text, tt.wantText)
			}
		})
	}
}
