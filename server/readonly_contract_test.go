package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// TestListClustersContract mirrors integration/test/cluster_test.go ("List all clusters").
func TestListClustersContract(t *testing.T) {
	clusterGET := func(resp string) exchange {
		return exchange{method: http.MethodGet, path: "/api/cluster?fields=*", resp: resp}
	}
	const (
		cdot  = `{"name":"c1","uuid":"u1","version":{"full":"NetApp Release 9.16.1","generation":9,"major":16,"minor":1},"san_optimized":false,"disaggregated":false}`
		afx   = `{"name":"c2","uuid":"u2","version":{"full":"NetApp Release 9.17.1","generation":9,"major":17,"minor":1},"san_optimized":false,"disaggregated":true}`
		asar2 = `{"name":"c3","uuid":"u3","version":{"full":"NetApp Release 9.18.1","generation":9,"major":18,"minor":1},"san_optimized":true,"disaggregated":true}`
	)

	tests := []struct {
		name     string
		order    []string
		scripts  map[string][]exchange
		wantText string // exact tool output
	}{
		{
			name:     "single cdot cluster",
			order:    []string{contractCluster},
			scripts:  map[string][]exchange{contractCluster: {clusterGET(cdot)}},
			wantText: `[{"name":"contract-cluster","ontap_version":"9.16","model":"cdot"}]`,
		},
		{
			name:  "clusters are sorted by name and report their model",
			order: []string{"zeta", "alpha", "mid"},
			scripts: map[string][]exchange{
				"zeta":  {clusterGET(cdot)},
				"alpha": {clusterGET(afx)},
				"mid":   {clusterGET(asar2)},
			},
			wantText: `[{"name":"alpha","ontap_version":"9.17","model":"afx"},` +
				`{"name":"mid","ontap_version":"9.18","model":"asar2"},` +
				`{"name":"zeta","ontap_version":"9.16","model":"cdot"}]`,
		},
		{
			// One unreachable or failing cluster must not hide the others.
			name:  "a failing cluster is listed without version",
			order: []string{"good", "bad"},
			scripts: map[string][]exchange{
				"good": {clusterGET(cdot)},
				"bad": {{method: http.MethodGet, path: "/api/cluster?fields=*", status: http.StatusUnauthorized,
					resp: `{"error":{"message":"not authorized for that command","code":"6"}}`}},
			},
			wantText: `[{"name":"bad","ontap_version":"","model":""},{"name":"good","ontap_version":"9.16","model":"cdot"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newOrderedClusterSession(t, "multiplex", tt.order, tt.scripts)
			text, isErr := callTool(t, cs, "list_registered_clusters", map[string]any{})
			assert.False(t, isErr)
			assert.Equal(t, text, tt.wantText)
		})
	}
}

// TestListQoSPoliciesContract mirrors integration/test/list_qos_policies_test.go: list all
// policies, filter by SVM (cluster-scoped policies must still be returned), and filter by
// fixed or adaptive type.
func TestListQoSPoliciesContract(t *testing.T) {
	const (
		svmFields      = "fields=*&max_records=500"
		fixedFields    = "class=user_defined&fields=policy_group,vserver,class,max_throughput,min_throughput,num_workloads,is_shared&max_records=500&vserver=cluster1"
		adaptiveFields = "fields=policy_group,vserver,expected_iops,peak_iops,absolute_min_iops,expected_iops_allocation,peak_iops_allocation,block_size,num_workloads&max_records=500&vserver=cluster1"

		svmFixed    = `{"name":"vtest-fixed","uuid":"p1","scope":"svm","object_count":1,"svm":{"name":"vtest","uuid":"s1"},"fixed":{"max_throughput_iops":5000,"capacity_shared":false}}`
		svmAdaptive = `{"name":"vtest-adaptive","uuid":"p2","scope":"svm","svm":{"name":"vtest","uuid":"s1"},"adaptive":{"expected_iops":100,"peak_iops":1000,"absolute_min_iops":10}}`
		cliFixed    = `{"num_records":1,"records":[{"policy_group":"gold","vserver":"cluster1","class":"user_defined","max_throughput":"1000IOPS,100MB/s","min_throughput":"0","num_workloads":2,"is_shared":true}]}`
		cliAdaptive = `{"num_records":1,"records":[{"policy_group":"extreme","vserver":"cluster1","expected_iops":"6144IOPS/TB","peak_iops":"12288IOPS/TB","absolute_min_iops":"1000IOPS","expected_iops_allocation":"allocated-space","peak_iops_allocation":"used-space","block_size":"ANY","num_workloads":0}]}`
	)
	svmPolicies := func(query, resp string) exchange {
		return exchange{method: http.MethodGet, path: "/api/storage/qos/policies?" + query, resp: resp}
	}
	adminSVM := exchange{method: http.MethodGet, path: "/api/private/cli/vserver?fields=vserver&max_records=1&type=admin", resp: `{"records":[{"vserver":"cluster1"}]}`}
	fixedPolicies := exchange{method: http.MethodGet, path: "/api/private/cli/qos/policy-group?" + fixedFields, resp: cliFixed}
	adaptivePolicies := exchange{method: http.MethodGet, path: "/api/private/cli/qos/adaptive-policy-group?" + adaptiveFields, resp: cliAdaptive}

	// The cluster-scoped policies as the tool reports them, after parsing the CLI throughput strings.
	gold := qoSPolicyRecord{Name: "gold", Scope: "cluster", ObjectCount: 2, SVM: qoSPolicySVM{Name: "cluster1"},
		Fixed: &qoSFixedLimits{MaxThroughputIOPS: 1000, MaxThroughputMbps: 100, CapacityShared: true}}
	extreme := qoSPolicyRecord{Name: "extreme", Scope: "cluster", SVM: qoSPolicySVM{Name: "cluster1"},
		Adaptive: &qoSAdaptiveLimits{ExpectedIOPS: 6144, PeakIOPS: 12288, AbsoluteMinIOPS: 1000, BlockSize: "any",
			ExpectedIOPSAllocation: "allocated-space", PeakIOPSAllocation: "used-space"}}
	vtestFixed := qoSPolicyRecord{Name: "vtest-fixed", UUID: "p1", Scope: "svm", ObjectCount: 1, SVM: qoSPolicySVM{Name: "vtest", UUID: "s1"},
		Fixed: &qoSFixedLimits{MaxThroughputIOPS: 5000}}
	vtestAdaptive := qoSPolicyRecord{Name: "vtest-adaptive", UUID: "p2", Scope: "svm", SVM: qoSPolicySVM{Name: "vtest", UUID: "s1"},
		Adaptive: &qoSAdaptiveLimits{ExpectedIOPS: 100, PeakIOPS: 1000, AbsoluteMinIOPS: 10}}

	tests := []struct {
		name     string
		args     map[string]any
		ontap    []exchange
		want     *QoSPoliciesResponse // decoded tool output, when the call succeeds
		wantText string               // substring of the output, when the call fails
	}{
		{
			name: "list all policies",
			args: map[string]any{"cluster_name": contractCluster},
			ontap: []exchange{
				svmPolicies(svmFields, `{"num_records":2,"records":[`+svmFixed+`,`+svmAdaptive+`]}`),
				adminSVM, fixedPolicies, adaptivePolicies,
			},
			want: &QoSPoliciesResponse{
				SVMPolicies:     []qoSPolicyRecord{vtestFixed, vtestAdaptive},
				ClusterPolicies: []qoSPolicyRecord{gold, extreme},
				NumRecords:      4,
			},
		},
		{
			name: "filter by svm still returns cluster-scoped policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "vtest"},
			ontap: []exchange{
				svmPolicies(svmFields+"&svm.name=vtest", `{"num_records":1,"records":[`+svmFixed+`]}`),
				adminSVM, fixedPolicies, adaptivePolicies,
			},
			want: &QoSPoliciesResponse{
				Message: "Showing 1 SVM-scoped policies for 'vtest' and 2 cluster-wide policies. Cluster-wide policies govern all workloads " +
					"on the cluster regardless of SVM and must be presented.",
				SVMPolicies:     []qoSPolicyRecord{vtestFixed},
				ClusterPolicies: []qoSPolicyRecord{gold, extreme},
				NumRecords:      3,
			},
		},
		{
			name: "filter by svm with no svm-scoped policies",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "vtest"},
			ontap: []exchange{
				svmPolicies(svmFields+"&svm.name=vtest", `{"num_records":0,"records":[]}`),
				adminSVM, fixedPolicies, adaptivePolicies,
			},
			want: &QoSPoliciesResponse{
				Message: "Showing 0 SVM-scoped policies for 'vtest' and 2 cluster-wide policies. Cluster-wide policies govern all workloads " +
					"on the cluster regardless of SVM and must be presented.",
				SVMPolicies:     []qoSPolicyRecord{},
				ClusterPolicies: []qoSPolicyRecord{gold, extreme},
				NumRecords:      2,
			},
		},
		{
			name: "fixed only skips adaptive lookup and filters svm policies",
			args: map[string]any{"cluster_name": contractCluster, "policy_type": "fixed"},
			ontap: []exchange{
				svmPolicies(svmFields, `{"num_records":2,"records":[`+svmFixed+`,`+svmAdaptive+`]}`),
				adminSVM, fixedPolicies,
			},
			want: &QoSPoliciesResponse{
				SVMPolicies:     []qoSPolicyRecord{vtestFixed},
				ClusterPolicies: []qoSPolicyRecord{gold},
				NumRecords:      2,
			},
		},
		{
			name: "adaptive only skips fixed lookup and filters svm policies",
			args: map[string]any{"cluster_name": contractCluster, "policy_type": "Adaptive"},
			ontap: []exchange{
				svmPolicies(svmFields, `{"num_records":2,"records":[`+svmFixed+`,`+svmAdaptive+`]}`),
				adminSVM, adaptivePolicies,
			},
			want: &QoSPoliciesResponse{
				SVMPolicies:     []qoSPolicyRecord{vtestAdaptive},
				ClusterPolicies: []qoSPolicyRecord{extreme},
				NumRecords:      2,
			},
		},
		{
			name: "follows pagination of svm-scoped policies",
			args: map[string]any{"cluster_name": contractCluster, "policy_type": "fixed"},
			ontap: []exchange{
				svmPolicies(svmFields, `{"num_records":1,"records":[`+svmFixed+`],"_links":{"next":{"href":"/api/storage/qos/policies?start.uuid=p1&fields=*&max_records=500"}}}`),
				svmPolicies("start.uuid=p1&fields=*&max_records=500", `{"num_records":1,"records":[`+svmAdaptive+`]}`),
				adminSVM, fixedPolicies,
			},
			want: &QoSPoliciesResponse{
				SVMPolicies:     []qoSPolicyRecord{vtestFixed},
				ClusterPolicies: []qoSPolicyRecord{gold},
				NumRecords:      2,
			},
		},
		{
			name:     "unsupported policy type is rejected without calling ONTAP",
			args:     map[string]any{"cluster_name": contractCluster, "policy_type": "throttle"},
			wantText: `unsupported policy_type "throttle": must be "fixed" or "adaptive"`,
		},
		{
			name:     "missing cluster is rejected without calling ONTAP",
			args:     map[string]any{"cluster_name": ""},
			wantText: "cluster_name is required",
		},
		{
			name:     "unknown cluster",
			args:     map[string]any{"cluster_name": "nope"},
			wantText: "cluster nope not found",
		},
		{
			name: "svm-scoped fetch surfaces ONTAP error",
			args: map[string]any{"cluster_name": contractCluster},
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/storage/qos/policies?" + svmFields, status: http.StatusForbidden,
					resp: `{"error":{"message":"not authorized for that command","code":"6"}}`},
			},
			wantText: "failed to fetch SVM-scoped qos policies: ErrValidator: message: not authorized for that command code: 6",
		},
		{
			name: "missing admin svm",
			args: map[string]any{"cluster_name": contractCluster},
			ontap: []exchange{
				svmPolicies(svmFields, `{"num_records":0,"records":[]}`),
				{method: http.MethodGet, path: "/api/private/cli/vserver?fields=vserver&max_records=1&type=admin", resp: `{"num_records":0,"records":[]}`},
			},
			wantText: "failed to get admin vserver: admin vserver not found",
		},
		{
			name: "cluster-scoped fixed fetch surfaces ONTAP error",
			args: map[string]any{"cluster_name": contractCluster},
			ontap: []exchange{
				svmPolicies(svmFields, `{"num_records":0,"records":[]}`),
				adminSVM,
				{method: http.MethodGet, path: "/api/private/cli/qos/policy-group?" + fixedFields, status: http.StatusBadRequest,
					resp: `{"error":{"message":"Unexpected argument \"class\".","code":"262179"}}`},
			},
			wantText: `failed to fetch cluster-scope fixed qos policies: ErrValidator: message: Unexpected argument "class". code: 262179`,
		},
		{
			name: "cluster-scoped adaptive fetch surfaces ONTAP error",
			args: map[string]any{"cluster_name": contractCluster, "policy_type": "adaptive"},
			ontap: []exchange{
				svmPolicies(svmFields, `{"num_records":0,"records":[]}`),
				adminSVM,
				{method: http.MethodGet, path: "/api/private/cli/qos/adaptive-policy-group?" + adaptiveFields, status: http.StatusForbidden,
					resp: `{"error":{"message":"not authorized for that command","code":"6"}}`},
			},
			wantText: "failed to fetch cluster-scope adaptive qos policies: ErrValidator: message: not authorized for that command code: 6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, "list_qos_policies", tt.args)
			assert.Equal(t, isErr, tt.want == nil)
			if tt.want == nil {
				if !strings.Contains(text, tt.wantText) {
					t.Errorf("tool output %q does not contain %q", text, tt.wantText)
				}
				return
			}
			var got QoSPoliciesResponse
			if err := json.Unmarshal([]byte(text), &got); err != nil {
				t.Fatalf("decoding tool output %q: %v", text, err)
			}
			assert.Equal(t, got, *tt.want)
			for _, p := range got.ClusterPolicies {
				if p.Scope != "cluster" {
					t.Errorf("cluster policy %s has scope %q, want cluster", p.Name, p.Scope)
				}
			}
			if pt, _ := tt.args["policy_type"].(string); pt != "" {
				key := `"` + strings.ToLower(pt) + `"`
				for _, p := range slices.Concat(got.SVMPolicies, got.ClusterPolicies) {
					raw, _ := json.Marshal(p)
					if !strings.Contains(string(raw), key) {
						t.Errorf("policy %s is not %s: %s", p.Name, pt, raw)
					}
				}
			}
		})
	}
}
