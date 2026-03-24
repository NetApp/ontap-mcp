package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"strconv"
	"strings"
)

func (a *App) CreateVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Volume) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	volumeCreate, err := newCreateVolume(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateVolume(ctx, volumeCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Volume created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Volume) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	volumeUpdate, err := newUpdateVolume(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateVolume(ctx, volumeUpdate, parameters.Volume, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Volume updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Volume) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	volumeDelete, err := newDeleteVolume(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteVolume(ctx, volumeDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Volume deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newDeleteVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteVolume(in tool.Volume) (ontap.Volume, error) {
	out := ontap.Volume{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Volume
	return out, nil
}

// newCreateVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateVolume(in tool.Volume) (ontap.Volume, error) {
	out := ontap.Volume{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Aggregate == "" {
		return out, errors.New("aggregate name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Aggregates = []ontap.NameAndUUID{
		{Name: in.Aggregate},
	}
	out.Name = in.Volume

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.Volume{}, err
		}
		out.Size = size
	}

	if in.ExportPolicy != "" || in.JunctionPath != "" {
		out.Nas = ontap.NAS{
			ExportPolicy: ontap.NASExportPolicy{
				Name: in.ExportPolicy,
			},
			Path: in.JunctionPath,
		}
	}

	switch {
	case in.QoS.RemovePolicy:
		out.QoS.Policy.Name = "none"
	case in.QoS.PolicyName != "":
		out.QoS.Policy.Name = in.QoS.PolicyName
	default:
		var err error
		if out.QoS.Policy.MaxThroughIOPS, err = parseQoSLimit(in.QoS.MaxIOPS); err != nil {
			return out, fmt.Errorf("invalid max_iops: %w", err)
		}
		if out.QoS.Policy.MinThroughIOPS, err = parseQoSLimit(in.QoS.MinIOPS); err != nil {
			return out, fmt.Errorf("invalid min_iops: %w", err)
		}
		if out.QoS.Policy.MaxThroughMBPS, err = parseQoSLimit(in.QoS.MaxMBPS); err != nil {
			return out, fmt.Errorf("invalid max_mbps: %w", err)
		}
		if out.QoS.Policy.MinThroughMBPS, err = parseQoSLimit(in.QoS.MinMBPS); err != nil {
			return out, fmt.Errorf("invalid min_mbps: %w", err)
		}
	}

	return out, nil
}

// newUpdateVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateVolume(in tool.Volume) (ontap.Volume, error) {
	out := ontap.Volume{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.NewVolume != "" {
		out.Name = in.NewVolume
	}

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.Volume{}, err
		}
		out.Size = size
	}

	if in.State != "" {
		out.State = in.State
	}

	if in.JunctionPath != "" {
		out.Nas.Path = in.JunctionPath
	}

	if in.ExportPolicy != "" {
		out.Nas.ExportPolicy.Name = in.ExportPolicy
	}

	if in.Autosize.Mode != "" {
		out.Autosize.Mode = in.Autosize.Mode
	}
	if in.Autosize.MaxSize != "" {
		out.Autosize.MaxSize = in.Autosize.MaxSize
	}
	if in.Autosize.MinSize != "" {
		out.Autosize.MinSize = in.Autosize.MinSize
	}
	if in.Autosize.GrowThreshold != "" {
		out.Autosize.GrowThreshold = in.Autosize.GrowThreshold
	}
	if in.Autosize.ShrinkThreshold != "" {
		out.Autosize.ShrinkThreshold = in.Autosize.ShrinkThreshold
	}

	switch {
	case in.QoS.RemovePolicy:
		out.QoS.Policy.Name = "none"
	case in.QoS.PolicyName != "":
		out.QoS.Policy.Name = in.QoS.PolicyName
	default:
		var err error
		if out.QoS.Policy.MaxThroughIOPS, err = parseQoSLimit(in.QoS.MaxIOPS); err != nil {
			return out, fmt.Errorf("invalid max_iops: %w", err)
		}
		if out.QoS.Policy.MinThroughIOPS, err = parseQoSLimit(in.QoS.MinIOPS); err != nil {
			return out, fmt.Errorf("invalid min_iops: %w", err)
		}
		if out.QoS.Policy.MaxThroughMBPS, err = parseQoSLimit(in.QoS.MaxMBPS); err != nil {
			return out, fmt.Errorf("invalid max_mbps: %w", err)
		}
		if out.QoS.Policy.MinThroughMBPS, err = parseQoSLimit(in.QoS.MinMBPS); err != nil {
			return out, fmt.Errorf("invalid min_mbps: %w", err)
		}
	}

	return out, nil
}

func parseQoSLimit(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 0 {
		return nil, fmt.Errorf("must be a non-negative integer, got %q", s)
	}
	return &v, nil
}
