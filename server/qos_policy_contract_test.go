package server

import (
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestQoSPolicyContract(t *testing.T) {
	const (
		policiesPath = "/api/storage/qos/policies"
		accepted     = `{"job":{"uuid":"qos-job"}}`
		noPolicies   = `{"num_records":0,"records":[]}`
	)
	jobDone := exchange{method: http.MethodGet, path: "/api/cluster/jobs/qos-job", resp: `{"uuid":"qos-job","state":"success"}`}

	// lookup is the name-to-UUID lookup that precedes every QoS policy update and delete; an
	// empty uuid means the policy does not exist.
	lookup := func(policy, uuid string) exchange {
		e := exchange{method: http.MethodGet, path: policiesPath + "?fields=uuid&svm=marketing&name=" + policy, resp: noPolicies}
		if uuid != "" {
			e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `"}]}`
		}
		return e
	}
	create := func(body string) exchange {
		return exchange{method: http.MethodPost, path: policiesPath, body: body, status: http.StatusAccepted, resp: accepted}
	}
	patch := func(uuid, body string) exchange {
		return exchange{method: http.MethodPatch, path: policiesPath + "/" + uuid, body: body, status: http.StatusAccepted, resp: accepted}
	}
	createArgs := func(name string, extra map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": name}
		maps.Copy(args, extra)
		return args
	}
	updateArgs := func(name string, update map[string]any) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "name": name, "qos_policy_update": update}
	}
	deleteArgs := func(name string) map[string]any {
		return map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "marketing", "name": name}
	}

	tests := []struct {
		name     string
		tool     string
		mode     string // tool mode, defaults to multiplex
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			name: "create fixed policy with only max iops",
			tool: "create_qos_policy",
			args: createArgs("gold", map[string]any{"max_throughput_iops": "5000"}),
			ontap: []exchange{
				create(`{"svm":{"name":"marketing"},"name":"gold","fixed":{"max_throughput_iops":5000,"min_throughput_iops":0,"capacity_shared":false}}`),
				jobDone,
			},
			wantText: "QoS Policy created successfully",
		},
		{
			name: "create fixed policy with max and min iops",
			tool: "create_qos_policy",
			args: createArgs("qos_docs_200iops", map[string]any{"max_throughput_iops": "200", "min_throughput_iops": "0"}),
			ontap: []exchange{
				{method: http.MethodPost, path: policiesPath, status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"qos_docs_200iops","fixed":{"max_throughput_iops":200,"min_throughput_iops":0,"capacity_shared":false}}`},
			},
			wantText: "QoS Policy created successfully",
		},
		{
			name: "create shared fixed policy",
			tool: "create_qos_policy",
			args: createArgs("gold", map[string]any{"max_throughput_iops": "5000", "min_throughput_iops": "100", "capacity_shared": true}),
			ontap: []exchange{
				create(`{"svm":{"name":"marketing"},"name":"gold","fixed":{"max_throughput_iops":5000,"min_throughput_iops":100,"capacity_shared":true}}`),
				jobDone,
			},
			wantText: "QoS Policy created successfully",
		},
		{
			name: "create adaptive policy",
			tool: "create_qos_policy",
			args: createArgs("payroll", map[string]any{"expected_iops": "2000", "peak_iops": "5000", "absolute_min_iops": "10"}),
			ontap: []exchange{
				create(`{"svm":{"name":"marketing"},"name":"payroll","adaptive":{"expected_iops":2000,"peak_iops":5000,"absolute_min_iops":10}}`),
				jobDone,
			},
			wantText: "QoS Policy created successfully",
		},
		{
			name: "create adaptive policy with allocation modes and block size",
			tool: "create_qos_policy",
			args: createArgs("alloc", map[string]any{
				"expected_iops": "1000", "peak_iops": "3000", "absolute_min_iops": "50",
				"expected_iops_allocation": "allocated_space", "peak_iops_allocation": "used_space", "block_size": "any",
			}),
			ontap: []exchange{
				create(`{"svm":{"name":"marketing"},"name":"alloc","adaptive":{"expected_iops":1000,"peak_iops":3000,"absolute_min_iops":50,
					"expected_iops_allocation":"allocated_space","peak_iops_allocation":"used_space","block_size":"any"}}`),
				jobDone,
			},
			wantText: "QoS Policy created successfully",
		},
		{
			// ONTAP 9.9 has no allocation fields; the integration suite expects this rejection.
			name: "create adaptive policy surfaces ONTAP unexpected argument",
			tool: "create_qos_policy",
			args: createArgs("alloc", map[string]any{
				"expected_iops": "1000", "peak_iops": "3000", "absolute_min_iops": "50", "expected_iops_allocation": "allocated_space",
			}),
			ontap: []exchange{
				{method: http.MethodPost, path: policiesPath, status: http.StatusBadRequest,
					body: `{"svm":{"name":"marketing"},"name":"alloc","adaptive":{"expected_iops":1000,"peak_iops":3000,"absolute_min_iops":50,"expected_iops_allocation":"allocated_space"}}`,
					resp: `{"error":{"message":"Unexpected argument \"expected_iops_allocation\".","code":"262179"}}`},
			},
			wantText: `Unexpected argument "expected_iops_allocation".`,
			wantErr:  true,
		},
		{
			name: "create surfaces async job failure",
			tool: "create_qos_policy",
			args: createArgs("gold", map[string]any{"max_throughput_iops": "5000"}),
			ontap: []exchange{
				create(`{"svm":{"name":"marketing"},"name":"gold","fixed":{"max_throughput_iops":5000,"min_throughput_iops":0,"capacity_shared":false}}`),
				{method: http.MethodGet, path: "/api/cluster/jobs/qos-job",
					resp: `{"uuid":"qos-job","state":"failure","error":{"message":"Policy group gold already exists","code":"8454147"}}`},
			},
			wantText: "Policy group gold already exists",
			wantErr:  true,
		},
		{
			name:     "create adaptive policy without peak iops is rejected without calling ONTAP",
			tool:     "create_qos_policy",
			args:     createArgs("payroll", map[string]any{"expected_iops": "2000", "absolute_min_iops": "10"}),
			wantText: "peak iops is required",
			wantErr:  true,
		},
		{
			name: "create adaptive policy with unknown allocation is rejected without calling ONTAP",
			tool: "create_qos_policy",
			args: createArgs("alloc", map[string]any{
				"expected_iops": "1000", "peak_iops": "3000", "absolute_min_iops": "50", "expected_iops_allocation": "free_space",
			}),
			wantText: `invalid expected_iops_allocation "free_space"`,
			wantErr:  true,
		},
		{
			name: "create adaptive policy with unknown block size is rejected without calling ONTAP",
			tool: "create_qos_policy",
			args: createArgs("alloc", map[string]any{
				"expected_iops": "1000", "peak_iops": "3000", "absolute_min_iops": "50", "block_size": "2048",
			}),
			wantText: `invalid block_size "2048"`,
			wantErr:  true,
		},
		{
			// IOPS are counts, not byte sizes: "5k" must not become 5120.
			name:     "create rejects iops with a unit suffix without calling ONTAP",
			tool:     "create_qos_policy",
			args:     createArgs("gold", map[string]any{"max_throughput_iops": "5k"}),
			wantText: `invalid max_throughput_iops "5k": must be a non-negative integer IOPS count without units`,
			wantErr:  true,
		},
		{
			name:     "create rejects negative iops without calling ONTAP",
			tool:     "create_qos_policy",
			args:     createArgs("gold", map[string]any{"max_throughput_iops": "5000", "min_throughput_iops": "-5"}),
			wantText: `invalid min_throughput_iops "-5": must be a non-negative integer IOPS count without units`,
			wantErr:  true,
		},
		{
			name:     "create rejects non-numeric adaptive iops without calling ONTAP",
			tool:     "create_qos_policy",
			args:     createArgs("payroll", map[string]any{"expected_iops": "fast", "peak_iops": "5000", "absolute_min_iops": "10"}),
			wantText: `invalid expected_iops "fast": must be a non-negative integer IOPS count without units`,
			wantErr:  true,
		},
		{
			name: "create rejects mixed fixed and adaptive fields without calling ONTAP",
			tool: "create_qos_policy",
			args: createArgs("gold", map[string]any{
				"max_throughput_iops": "5000", "expected_iops": "2000", "peak_iops": "5000", "absolute_min_iops": "10",
			}),
			wantText: "a QoS policy is either fixed or adaptive: fixed fields (max_throughput_iops) cannot be combined with adaptive fields (expected_iops, peak_iops, absolute_min_iops)",
			wantErr:  true,
		},
		{
			// capacity_shared applies only to fixed policies.
			name:     "create rejects shared adaptive policy without calling ONTAP",
			tool:     "create_qos_policy",
			args:     createArgs("payroll", map[string]any{"capacity_shared": true, "expected_iops": "2000", "peak_iops": "5000", "absolute_min_iops": "10"}),
			wantText: "fixed fields (capacity_shared) cannot be combined with adaptive fields (expected_iops, peak_iops, absolute_min_iops)",
			wantErr:  true,
		},
		{
			name: "rename policy by UUID",
			tool: "modify_qos_policy",
			args: updateArgs("gold", map[string]any{"new_name": "silver"}),
			ontap: []exchange{
				lookup("gold", "qos-uuid"),
				patch("qos-uuid", `{"name":"silver"}`),
				jobDone,
			},
			wantText: "QoS policy updated successfully",
		},
		{
			name: "update adaptive allocation modes only",
			tool: "modify_qos_policy",
			args: updateArgs("alloc", map[string]any{"expected_iops_allocation": "allocated_space", "peak_iops_allocation": "allocated_space"}),
			ontap: []exchange{
				lookup("alloc", "qos-uuid"),
				{method: http.MethodPatch, path: policiesPath + "/qos-uuid",
					body: `{"adaptive":{"expected_iops_allocation":"allocated_space","peak_iops_allocation":"allocated_space"}}`},
			},
			wantText: "QoS policy updated successfully",
		},
		{
			name: "update adaptive iops",
			tool: "modify_qos_policy",
			args: updateArgs("payroll", map[string]any{"expected_iops": "3000", "peak_iops": "6000", "absolute_min_iops": "20"}),
			ontap: []exchange{
				lookup("payroll", "qos-uuid"),
				patch("qos-uuid", `{"adaptive":{"expected_iops":3000,"peak_iops":6000,"absolute_min_iops":20}}`),
				jobDone,
			},
			wantText: "QoS policy updated successfully",
		},
		{
			// An update sends only the fields provided: no capacity_shared, which could unshare a shared policy.
			name: "update fixed iops sends only max and min",
			tool: "modify_qos_policy",
			args: updateArgs("gold", map[string]any{"max_throughput_iops": "100", "min_throughput_iops": "0"}),
			ontap: []exchange{
				lookup("gold", "qos-uuid"),
				patch("qos-uuid", `{"fixed":{"max_throughput_iops":100,"min_throughput_iops":0}}`),
				jobDone,
			},
			wantText: "QoS policy updated successfully",
		},
		{
			name: "legacy update fixed iops sends only max and min",
			tool: "update_qos_policy",
			mode: "legacy",
			args: createArgs("gold", map[string]any{"max_throughput_iops": "100", "min_throughput_iops": "0"}),
			ontap: []exchange{
				lookup("gold", "qos-uuid"),
				patch("qos-uuid", `{"fixed":{"max_throughput_iops":100,"min_throughput_iops":0}}`),
				jobDone,
			},
			wantText: "QoS Policy updated successfully",
		},
		{
			// The integration suite's allocation update on a 9.9 cluster depends on this message.
			name:     "update of missing policy",
			tool:     "modify_qos_policy",
			args:     updateArgs("alloc", map[string]any{"expected_iops_allocation": "allocated_space", "peak_iops_allocation": "allocated_space"}),
			ontap:    []exchange{lookup("alloc", "")},
			wantText: "failed to update qos policy alloc on svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "update surfaces ONTAP error",
			tool: "modify_qos_policy",
			args: updateArgs("gold", map[string]any{"new_name": "silver"}),
			ontap: []exchange{
				lookup("gold", "qos-uuid"),
				{method: http.MethodPatch, path: policiesPath + "/qos-uuid", body: `{"name":"silver"}`, status: http.StatusConflict,
					resp: `{"error":{"message":"Policy group silver already exists","code":"8454147"}}`},
			},
			wantText: "Policy group silver already exists",
			wantErr:  true,
		},
		{
			name:     "update fixed policy with only max iops is rejected without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     updateArgs("gold", map[string]any{"max_throughput_iops": "1000"}),
			wantText: "both max_throughput_iops and min_throughput_iops must be provided for fixed QoS policy",
			wantErr:  true,
		},
		{
			name:     "update adaptive policy with partial iops is rejected without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     updateArgs("payroll", map[string]any{"expected_iops": "3000", "peak_iops": "6000"}),
			wantText: "all of expected_iops, peak_iops, and absolute_min_iops must be provided for adaptive QoS policy",
			wantErr:  true,
		},
		{
			name:     "update with unknown allocation is rejected without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     updateArgs("alloc", map[string]any{"peak_iops_allocation": "free_space"}),
			wantText: `invalid peak_iops_allocation "free_space"`,
			wantErr:  true,
		},
		{
			name:     "update rejects iops with a unit suffix without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     updateArgs("gold", map[string]any{"max_throughput_iops": "5k", "min_throughput_iops": "0"}),
			wantText: `invalid max_throughput_iops "5k": must be a non-negative integer IOPS count without units`,
			wantErr:  true,
		},
		{
			name:     "update rejects negative iops without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     updateArgs("payroll", map[string]any{"expected_iops": "3000", "peak_iops": "-5", "absolute_min_iops": "20"}),
			wantText: `invalid peak_iops "-5": must be a non-negative integer IOPS count without units`,
			wantErr:  true,
		},
		{
			name:     "update rejects non-numeric iops without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     updateArgs("payroll", map[string]any{"expected_iops": "3000", "peak_iops": "6000", "absolute_min_iops": "fast"}),
			wantText: `invalid absolute_min_iops "fast": must be a non-negative integer IOPS count without units`,
			wantErr:  true,
		},
		{
			name: "update rejects mixed fixed and adaptive fields without calling ONTAP",
			tool: "modify_qos_policy",
			args: updateArgs("gold", map[string]any{
				"max_throughput_iops": "100", "min_throughput_iops": "0", "expected_iops_allocation": "allocated_space",
			}),
			wantText: "a QoS policy is either fixed or adaptive: fixed fields (max_throughput_iops, min_throughput_iops) cannot be combined with adaptive fields (expected_iops_allocation)",
			wantErr:  true,
		},
		{
			name:     "update with no fields is rejected without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "marketing", "name": "gold"},
			wantText: "at least one updatable field must be provided",
			wantErr:  true,
		},
		{
			name: "delete policy by UUID",
			tool: "modify_qos_policy",
			args: deleteArgs("silver"),
			ontap: []exchange{
				lookup("silver", "qos-uuid"),
				{method: http.MethodDelete, path: policiesPath + "/qos-uuid", status: http.StatusAccepted, resp: accepted},
				jobDone,
			},
			wantText: "QoS policy deleted successfully",
		},
		{
			// The integration suite's "Clean QoS policy" steps depend on this message.
			name:     "delete of missing policy",
			tool:     "modify_qos_policy",
			args:     deleteArgs("gold"),
			ontap:    []exchange{lookup("gold", "")},
			wantText: "failed to delete qos policy gold on svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete surfaces ONTAP error",
			tool: "modify_qos_policy",
			args: deleteArgs("gold"),
			ontap: []exchange{
				lookup("gold", "qos-uuid"),
				{method: http.MethodDelete, path: policiesPath + "/qos-uuid", status: http.StatusBadRequest,
					resp: `{"error":{"message":"Cannot delete policy group gold because it is in use","code":"8454149"}}`},
			},
			wantText: "Cannot delete policy group gold because it is in use",
			wantErr:  true,
		},
		{
			name:     "modify without SVM is rejected without calling ONTAP",
			tool:     "modify_qos_policy",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "", "name": "gold"},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name:     "modify rejects unknown operation",
			tool:     "modify_qos_policy",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "rename", "svm_name": "marketing", "name": "gold"},
			wantText: `unsupported operation "rename"`,
			wantErr:  true,
		},
		{
			name:     "unknown cluster",
			tool:     "create_qos_policy",
			args:     map[string]any{"cluster_name": "nope", "svm_name": "marketing", "name": "gold", "max_throughput_iops": "5000"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := tt.mode
			if mode == "" {
				mode = "multiplex"
			}
			cs := newContractSession(t, mode, tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("got: %q; want to contain: %q", text, tt.wantText)
			}
		})
	}
}
