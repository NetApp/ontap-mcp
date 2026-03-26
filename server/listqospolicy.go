package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListQoSPoliciesParams struct {
	Cluster    string `json:"cluster_name" jsonschema:"cluster name, from list_registered_clusters"`
	SVMName    string `json:"svm_name,omitzero" jsonschema:"filters SVM-scoped policies to this SVM; cluster-scoped policies apply to all SVMs and must always be shown in the response"`
	PolicyType string `json:"policy_type,omitzero" jsonschema:"filter by policy type: fixed or adaptive"`
}
type qoSPolicySVM struct {
	Name string `json:"name"`
	UUID string `json:"uuid,omitempty"`
}

type qoSFixedLimits struct {
	MaxThroughputIOPS int64   `json:"max_throughput_iops,omitempty"`
	MaxThroughputMbps float64 `json:"max_throughput_mbps,omitempty"`
	MinThroughputIOPS int64   `json:"min_throughput_iops,omitempty"`
	MinThroughputMbps float64 `json:"min_throughput_mbps,omitempty"`
	CapacityShared    bool    `json:"capacity_shared"`
}

type qoSAdaptiveLimits struct {
	ExpectedIOPS           int64  `json:"expected_iops,omitempty"`
	PeakIOPS               int64  `json:"peak_iops,omitempty"`
	AbsoluteMinIOPS        int64  `json:"absolute_min_iops,omitempty"`
	BlockSize              string `json:"block_size,omitempty"`
	ExpectedIOPSAllocation string `json:"expected_iops_allocation,omitempty"`
	PeakIOPSAllocation     string `json:"peak_iops_allocation,omitempty"`
}

type qoSPolicyRecord struct {
	Name        string             `json:"name"`
	UUID        string             `json:"uuid,omitempty"`
	Scope       string             `json:"scope,omitempty"`
	ObjectCount int                `json:"object_count,omitempty"`
	SVM         qoSPolicySVM       `json:"svm"`
	Fixed       *qoSFixedLimits    `json:"fixed,omitempty"`
	Adaptive    *qoSAdaptiveLimits `json:"adaptive,omitempty"`
}

type QoSPoliciesResponse struct {
	Message         string            `json:"message,omitempty"`
	SVMPolicies     []qoSPolicyRecord `json:"svm_policies"`
	ClusterPolicies []qoSPolicyRecord `json:"cluster_policies"`
	NumRecords      int               `json:"num_records"`
}

type listQoSPoliciesArgs struct {
	cluster    string
	svmName    string
	policyType string
}

func newListQoSPolicies(p ListQoSPoliciesParams) (listQoSPoliciesArgs, error) {
	if p.Cluster == "" {
		return listQoSPoliciesArgs{}, errors.New("cluster_name is required")
	}
	pt := strings.ToLower(strings.TrimSpace(p.PolicyType))
	if pt != "" && pt != "fixed" && pt != "adaptive" {
		return listQoSPoliciesArgs{}, fmt.Errorf("unsupported policy_type %q: must be \"fixed\" or \"adaptive\"", p.PolicyType)
	}
	return listQoSPoliciesArgs{cluster: p.Cluster, svmName: p.SVMName, policyType: pt}, nil
}

func (a *App) ListQoSPolicies(ctx context.Context, _ *mcp.CallToolRequest, p ListQoSPoliciesParams) (*mcp.CallToolResult, QoSPoliciesResponse, error) {
	empty := QoSPoliciesResponse{}
	args, err := newListQoSPolicies(p)
	if err != nil {
		return errorResult(err), empty, nil
	}

	a.locks.RLock(args.cluster)
	defer a.locks.RUnlock(args.cluster)

	client, err := a.getClient(args.cluster)
	if err != nil {
		return errorResult(err), empty, err
	}

	svmRecords, err := client.GetSVMQoSPolicies(ctx, args.svmName)
	if err != nil {
		return errorResult(fmt.Errorf("failed to fetch SVM-scoped qos policies: %w", err)), empty, err
	}

	adminSVM, err := client.GetAdminSVM(ctx)
	if err != nil {
		return errorResult(fmt.Errorf("failed to get admin vserver: %w", err)), empty, err
	}

	var adminRecords []json.RawMessage
	if args.policyType == "" || args.policyType == "fixed" {
		fixed, err := client.GetAdminSVMFixedPolicies(ctx, adminSVM)
		if err != nil {
			return errorResult(fmt.Errorf("failed to fetch cluster-scope fixed qos policies: %w", err)), empty, err
		}
		adminRecords = append(adminRecords, fixed...)
	}
	if args.policyType == "" || args.policyType == "adaptive" {
		adaptive, err := client.GetAdminSVMAdaptivePolicies(ctx, adminSVM)
		if err != nil {
			return errorResult(fmt.Errorf("failed to fetch cluster-scope adaptive qos policies: %w", err)), empty, err
		}
		adminRecords = append(adminRecords, adaptive...)
	}

	if args.policyType != "" {
		svmRecords = filterRecordsByType(svmRecords, args.policyType)
	}

	svmPolicies := make([]qoSPolicyRecord, 0, len(svmRecords))
	for _, raw := range svmRecords {
		var rec qoSPolicyRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			slog.Warn("skipping qos policy record: failed to unmarshal", slog.Any("error", err))
			continue
		}
		svmPolicies = append(svmPolicies, rec)
	}

	clusterPolicies := make([]qoSPolicyRecord, 0, len(adminRecords))
	for _, raw := range adminRecords {
		var rec qoSPolicyRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			slog.Warn("skipping qos policy record: failed to unmarshal", slog.Any("error", err))
			continue
		}
		clusterPolicies = append(clusterPolicies, rec)
	}

	resp := QoSPoliciesResponse{
		SVMPolicies:     svmPolicies,
		ClusterPolicies: clusterPolicies,
		NumRecords:      len(svmPolicies) + len(clusterPolicies),
	}
	if args.svmName != "" {
		resp.Message = fmt.Sprintf(
			"Showing %d SVM-scoped policies for '%s' and %d cluster-wide policies. Cluster-wide policies govern all workloads on the cluster regardless of SVM and must be presented.",
			len(svmPolicies), args.svmName, len(clusterPolicies),
		)
	}
	return nil, resp, nil
}

func filterRecordsByType(records []json.RawMessage, policyType string) []json.RawMessage {
	var result []json.RawMessage
	wantKey := strings.ToLower(policyType)
	for _, rec := range records {
		var check map[string]json.RawMessage
		if err := json.Unmarshal(rec, &check); err != nil {
			continue
		}
		if _, ok := check[wantKey]; ok {
			result = append(result, rec)
		}
	}
	return result
}
