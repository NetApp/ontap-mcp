package server

import (
	"context"
	"errors"
	"fmt"

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

func updateQoSPolicyValidation(in tool.QoSPolicyUpdate) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}

	hasUpdate := false
	if in.NewName != "" {
		out.Name = in.NewName
		hasUpdate = true
	}

	if in.MaxThIOPS != "" || in.MinThIOPS != "" {
		if in.MaxThIOPS == "" || in.MinThIOPS == "" {
			return out, errors.New("both max_throughput_iops and min_throughput_iops must be provided for fixed QoS policy")
		}
		maxiops, err := parseSize(in.MaxThIOPS)
		if err != nil {
			return out, fmt.Errorf("invalid max_throughput_iops: %w", err)
		}
		miniops, err := parseSize(in.MinThIOPS)
		if err != nil {
			return out, fmt.Errorf("invalid min_throughput_iops: %w", err)
		}
		out.Fixed = ontap.QoSFixed{
			MaxThIOPS: maxiops,
			MinThIOPS: miniops,
		}
		hasUpdate = true
	} else if in.ExpectedIOPS != "" || in.PeakIOPS != "" || in.AbsoluteMinIOPS != "" ||
		in.ExpectedIOPSAllocation != "" || in.PeakIOPSAllocation != "" || in.BlockSize != "" {
		// IOPS fields are all-or-nothing; allocation/block_size may be updated independently.
		hasIOPS := in.ExpectedIOPS != "" || in.PeakIOPS != "" || in.AbsoluteMinIOPS != ""
		if hasIOPS {
			if in.ExpectedIOPS == "" || in.PeakIOPS == "" || in.AbsoluteMinIOPS == "" {
				return out, errors.New("all of expected_iops, peak_iops, and absolute_min_iops must be provided for adaptive QoS policy")
			}
			expectediops, err := parseSize(in.ExpectedIOPS)
			if err != nil {
				return out, fmt.Errorf("invalid expected_iops: %w", err)
			}
			peakiops, err := parseSize(in.PeakIOPS)
			if err != nil {
				return out, fmt.Errorf("invalid peak_iops: %w", err)
			}
			absoluteMiniops, err := parseSize(in.AbsoluteMinIOPS)
			if err != nil {
				return out, fmt.Errorf("invalid absolute_min_iops: %w", err)
			}
			out.Adaptive.ExpectedIOPS = expectediops
			out.Adaptive.PeakIOPS = peakiops
			out.Adaptive.AbsoluteMinIOPS = absoluteMiniops
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

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	if in.MaxThIOPS != "" || in.MinThIOPS != "" {
		// Only one of these need to be provided to create a fixed qos policy

		maxiops, err := parseSizeEmptyAllowed(in.MaxThIOPS)
		if err != nil {
			return out, err
		}
		miniops, err := parseSizeEmptyAllowed(in.MinThIOPS)
		if err != nil {
			return out, err
		}
		out.Fixed = ontap.QoSFixed{
			MaxThIOPS:      maxiops,
			MinThIOPS:      miniops,
			CapacityShared: in.CapacityShared,
		}
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

		expectediops, err := parseSizeEmptyAllowed(in.ExpectedIOPS)
		if err != nil {
			return out, err
		}
		peakiops, err := parseSizeEmptyAllowed(in.PeakIOPS)
		if err != nil {
			return out, err
		}
		absoluteMiniops, err := parseSizeEmptyAllowed(in.AbsoluteMinIOPS)
		if err != nil {
			return out, err
		}
		if err := validateAdaptiveAllocationFields(in.ExpectedIOPSAllocation, in.PeakIOPSAllocation, in.BlockSize); err != nil {
			return out, err
		}
		out.Adaptive = ontap.QoSAdaptive{
			ExpectedIOPS:           expectediops,
			PeakIOPS:               peakiops,
			AbsoluteMinIOPS:        absoluteMiniops,
			ExpectedIOPSAllocation: in.ExpectedIOPSAllocation,
			PeakIOPSAllocation:     in.PeakIOPSAllocation,
			BlockSize:              in.BlockSize,
		}
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

		maxiops, err := parseSize(in.MaxThIOPS)
		if err != nil {
			return out, err
		}
		miniops, err := parseSize(in.MinThIOPS)
		if err != nil {
			return out, err
		}
		out.Fixed = ontap.QoSFixed{
			MaxThIOPS: maxiops,
			MinThIOPS: miniops,
		}
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

			expectediops, err := parseSize(in.ExpectedIOPS)
			if err != nil {
				return out, err
			}
			peakiops, err := parseSize(in.PeakIOPS)
			if err != nil {
				return out, err
			}
			absoluteMiniops, err := parseSize(in.AbsoluteMinIOPS)
			if err != nil {
				return out, err
			}
			out.Adaptive.ExpectedIOPS = expectediops
			out.Adaptive.PeakIOPS = peakiops
			out.Adaptive.AbsoluteMinIOPS = absoluteMiniops
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
