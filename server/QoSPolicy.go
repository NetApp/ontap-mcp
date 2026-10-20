package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

var validIOPSAllocations = map[string]bool{"allocated_space": true, "used_space": true}
var validBlockSizes = map[string]bool{"any": true, "512": true, "1024": true, "4096": true}

func validateAdaptiveAllocationFields(expectedAlloc, peakAlloc, blockSize string) error {
	if expectedAlloc != "" && !validIOPSAllocations[expectedAlloc] {
		return fmt.Errorf("invalid expected_iops_allocation %q; valid values are allocated_space, used_space", expectedAlloc)
	}
	if peakAlloc != "" && !validIOPSAllocations[peakAlloc] {
		return fmt.Errorf("invalid peak_iops_allocation %q; valid values are allocated_space, used_space", peakAlloc)
	}
	if blockSize != "" && !validBlockSizes[blockSize] {
		return fmt.Errorf("invalid block_size %q; valid values are any, 512, 1024, 4096", blockSize)
	}
	return nil
}

func (a *App) CreateQoSPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicy) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	qosPolicyCreate, err := newCreateQoSPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateQoSPolicy(ctx, qosPolicyCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "QoS Policy created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateQosPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicy) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	qosPolicyUpdate, err := newUpdateQoSPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateQoSPolicy(ctx, qosPolicyUpdate, parameters.Name, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "QoS Policy updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteQoSPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicy) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	qosPolicyDelete, err := newDeleteQoSPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteQoSPolicy(ctx, qosPolicyDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "QoS policy deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) ModifyQoSPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicyModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}
	if parameters.Name == "" {
		return nil, nil, errors.New("QoS policy name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		qosPolicyUpdate, err := updateQoSPolicyValidation(parameters.QoSPolicyUpdate)
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateQoSPolicy(ctx, qosPolicyUpdate, parameters.Name, parameters.SVM)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "QoS policy updated successfully"},
			},
		}, nil, nil
	case "delete":
		qosPolicyDelete := ontap.QoSPolicy{
			SVM:  ontap.NameAndUUID{Name: parameters.SVM},
			Name: parameters.Name,
		}

		err = client.DeleteQoSPolicy(ctx, qosPolicyDelete)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "QoS policy deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

// parseQoSPolicyIOPS parses a QoS policy IOPS value. IOPS are counts, not byte sizes, so only
// plain non-negative integers are accepted; a suffix such as "5k" is rejected rather than scaled.
// An empty string means the field was not provided and yields nil.
func parseQoSPolicyIOPS(field, s string) (*int64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || v < 0 {
		return nil, fmt.Errorf("invalid %s %q: must be a non-negative integer IOPS count without units", field, s)
	}
	return &v, nil
}

// checkQoSPolicyType rejects a request that mixes fixed and adaptive fields. A QoS policy is
// either fixed or adaptive, so one set of fields would otherwise be silently dropped.
func checkQoSPolicyType(maxIOPS, minIOPS string, capacityShared bool,
	expectedIOPS, peakIOPS, absoluteMinIOPS, expectedAlloc, peakAlloc, blockSize string) error {
	var fixed, adaptive []string
	add := func(list []string, set bool, name string) []string {
		if set {
			return append(list, name)
		}
		return list
	}
	fixed = add(fixed, maxIOPS != "", "max_throughput_iops")
	fixed = add(fixed, minIOPS != "", "min_throughput_iops")
	fixed = add(fixed, capacityShared, "capacity_shared")
	adaptive = add(adaptive, expectedIOPS != "", "expected_iops")
	adaptive = add(adaptive, peakIOPS != "", "peak_iops")
	adaptive = add(adaptive, absoluteMinIOPS != "", "absolute_min_iops")
	adaptive = add(adaptive, expectedAlloc != "", "expected_iops_allocation")
	adaptive = add(adaptive, peakAlloc != "", "peak_iops_allocation")
	adaptive = add(adaptive, blockSize != "", "block_size")
	if len(fixed) == 0 || len(adaptive) == 0 {
		return nil
	}
	return fmt.Errorf("a QoS policy is either fixed or adaptive: fixed fields (%s) cannot be combined with adaptive fields (%s)",
		strings.Join(fixed, ", "), strings.Join(adaptive, ", "))
}

// parseAdaptiveIOPS parses the three adaptive IOPS fields into out.
func parseAdaptiveIOPS(out *ontap.QoSAdaptive, expectedIOPS, peakIOPS, absoluteMinIOPS string) error {
	var err error
	if out.ExpectedIOPS, err = parseQoSPolicyIOPS("expected_iops", expectedIOPS); err != nil {
		return err
	}
	if out.PeakIOPS, err = parseQoSPolicyIOPS("peak_iops", peakIOPS); err != nil {
		return err
	}
	if out.AbsoluteMinIOPS, err = parseQoSPolicyIOPS("absolute_min_iops", absoluteMinIOPS); err != nil {
		return err
	}
	return nil
}

// parseFixedIOPS parses the fixed max/min IOPS fields. capacity_shared is left unset because it
// can only be chosen when the policy is created.
func parseFixedIOPS(maxIOPS, minIOPS string) (ontap.QoSFixed, error) {
	var (
		out ontap.QoSFixed
		err error
	)
	if out.MaxThIOPS, err = parseQoSPolicyIOPS("max_throughput_iops", maxIOPS); err != nil {
		return out, err
	}
	if out.MinThIOPS, err = parseQoSPolicyIOPS("min_throughput_iops", minIOPS); err != nil {
		return out, err
	}
	return out, nil
}

func updateQoSPolicyValidation(in tool.QoSPolicyUpdate) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}

	if err := checkQoSPolicyType(in.MaxThIOPS, in.MinThIOPS, false, in.ExpectedIOPS, in.PeakIOPS, in.AbsoluteMinIOPS,
		in.ExpectedIOPSAllocation, in.PeakIOPSAllocation, in.BlockSize); err != nil {
		return out, err
	}

	hasUpdate := false
	if in.NewName != "" {
		out.Name = in.NewName
		hasUpdate = true
	}

	if in.MaxThIOPS != "" || in.MinThIOPS != "" {
		if in.MaxThIOPS == "" || in.MinThIOPS == "" {
			return out, errors.New("both max_throughput_iops and min_throughput_iops must be provided for fixed QoS policy")
		}
		fixed, err := parseFixedIOPS(in.MaxThIOPS, in.MinThIOPS)
		if err != nil {
			return out, err
		}
		out.Fixed = fixed
		hasUpdate = true
	} else if in.ExpectedIOPS != "" || in.PeakIOPS != "" || in.AbsoluteMinIOPS != "" ||
		in.ExpectedIOPSAllocation != "" || in.PeakIOPSAllocation != "" || in.BlockSize != "" {
		// IOPS fields are all-or-nothing; allocation/block_size may be updated independently.
		hasIOPS := in.ExpectedIOPS != "" || in.PeakIOPS != "" || in.AbsoluteMinIOPS != ""
		if hasIOPS {
			if in.ExpectedIOPS == "" || in.PeakIOPS == "" || in.AbsoluteMinIOPS == "" {
				return out, errors.New("all of expected_iops, peak_iops, and absolute_min_iops must be provided for adaptive QoS policy")
			}
			if err := parseAdaptiveIOPS(&out.Adaptive, in.ExpectedIOPS, in.PeakIOPS, in.AbsoluteMinIOPS); err != nil {
				return out, err
			}
		}
		if err := validateAdaptiveAllocationFields(in.ExpectedIOPSAllocation, in.PeakIOPSAllocation, in.BlockSize); err != nil {
			return out, err
		}
		out.Adaptive.ExpectedIOPSAllocation = in.ExpectedIOPSAllocation
		out.Adaptive.PeakIOPSAllocation = in.PeakIOPSAllocation
		out.Adaptive.BlockSize = in.BlockSize
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name, max_throughput_iops & min_throughput_iops, expected_iops & peak_iops & absolute_min_iops, or expected_iops_allocation/peak_iops_allocation/block_size")
	}

	return out, nil
}

// newCreateQoSPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateQoSPolicy(in tool.QoSPolicy) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("qos policy name is required")
	}
	if err := checkQoSPolicyType(in.MaxThIOPS, in.MinThIOPS, in.CapacityShared, in.ExpectedIOPS, in.PeakIOPS, in.AbsoluteMinIOPS,
		in.ExpectedIOPSAllocation, in.PeakIOPSAllocation, in.BlockSize); err != nil {
		return out, err
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	if in.MaxThIOPS != "" || in.MinThIOPS != "" {
		// Only one of these need to be provided to create a fixed qos policy; the other defaults to 0.
		fixed, err := parseFixedIOPS(in.MaxThIOPS, in.MinThIOPS)
		if err != nil {
			return out, err
		}
		if fixed.MaxThIOPS == nil {
			fixed.MaxThIOPS = new(int64)
		}
		if fixed.MinThIOPS == nil {
			fixed.MinThIOPS = new(int64)
		}
		fixed.CapacityShared = &in.CapacityShared
		out.Fixed = fixed
	} else {
		if in.ExpectedIOPS == "" {
			return out, errors.New("expected iops is required")
		}
		if in.PeakIOPS == "" {
			return out, errors.New("peak iops is required")
		}
		if in.AbsoluteMinIOPS == "" {
			return out, errors.New("absolute min iops is required")
		}

		if err := parseAdaptiveIOPS(&out.Adaptive, in.ExpectedIOPS, in.PeakIOPS, in.AbsoluteMinIOPS); err != nil {
			return out, err
		}
		if err := validateAdaptiveAllocationFields(in.ExpectedIOPSAllocation, in.PeakIOPSAllocation, in.BlockSize); err != nil {
			return out, err
		}
		out.Adaptive.ExpectedIOPSAllocation = in.ExpectedIOPSAllocation
		out.Adaptive.PeakIOPSAllocation = in.PeakIOPSAllocation
		out.Adaptive.BlockSize = in.BlockSize
	}

	return out, nil
}

// newUpdateQoSPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateQoSPolicy(in tool.QoSPolicy) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" && in.NewName == "" {
		return out, errors.New("qos policy name is required")
	}
	// capacity_shared is not checked here: it can only be set at create time and is never sent on update.
	if err := checkQoSPolicyType(in.MaxThIOPS, in.MinThIOPS, false, in.ExpectedIOPS, in.PeakIOPS, in.AbsoluteMinIOPS,
		in.ExpectedIOPSAllocation, in.PeakIOPSAllocation, in.BlockSize); err != nil {
		return out, err
	}

	hasUpdate := false
	if in.NewName != "" {
		out.Name = in.NewName
		hasUpdate = true
	}

	if in.MaxThIOPS != "" || in.MinThIOPS != "" {
		if in.MaxThIOPS == "" {
			return out, errors.New("max throughput iops is required")
		}
		if in.MinThIOPS == "" {
			return out, errors.New("min throughput iops is required")
		}

		fixed, err := parseFixedIOPS(in.MaxThIOPS, in.MinThIOPS)
		if err != nil {
			return out, err
		}
		out.Fixed = fixed
		hasUpdate = true
	} else if in.ExpectedIOPS != "" || in.PeakIOPS != "" || in.AbsoluteMinIOPS != "" ||
		in.ExpectedIOPSAllocation != "" || in.PeakIOPSAllocation != "" || in.BlockSize != "" {
		// IOPS fields are all-or-nothing; allocation/block_size may be updated independently.
		hasIOPS := in.ExpectedIOPS != "" || in.PeakIOPS != "" || in.AbsoluteMinIOPS != ""
		if hasIOPS {
			if in.ExpectedIOPS == "" {
				return out, errors.New("expected iops is required")
			}
			if in.PeakIOPS == "" {
				return out, errors.New("peak iops is required")
			}
			if in.AbsoluteMinIOPS == "" {
				return out, errors.New("absolute min iops is required")
			}

			if err := parseAdaptiveIOPS(&out.Adaptive, in.ExpectedIOPS, in.PeakIOPS, in.AbsoluteMinIOPS); err != nil {
				return out, err
			}
		}
		if err := validateAdaptiveAllocationFields(in.ExpectedIOPSAllocation, in.PeakIOPSAllocation, in.BlockSize); err != nil {
			return out, err
		}
		out.Adaptive.ExpectedIOPSAllocation = in.ExpectedIOPSAllocation
		out.Adaptive.PeakIOPSAllocation = in.PeakIOPSAllocation
		out.Adaptive.BlockSize = in.BlockSize
		hasUpdate = true
	}
	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name, max_throughput_iops & min_throughput_iops, expected_iops & peak_iops & absolute_min_iops, or expected_iops_allocation/peak_iops_allocation/block_size")
	}

	return out, nil
}

// newDeleteQoSPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteQoSPolicy(in tool.QoSPolicy) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("qos policy name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	return out, nil
}
