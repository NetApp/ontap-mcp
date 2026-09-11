package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"log/slog"
	"strconv"
	"strings"
)

func (a *App) CreateVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.VolumeCreate) (*mcp.CallToolResult, any, error) {
	model := ontap.CDOT
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if _, m, err := a.getClusterVersion(ctx, parameters.Cluster); err == nil {
		model = m
	} else {
		a.logger.Warn("failed to determine cluster model, choosing default model as CDOT", slog.String("cluster", parameters.Cluster), slog.String("error", err.Error()))
	}

	remote := ontap.Remote{Model: model}
	if info, err := client.Remote(ctx); err == nil && !info.IsZero() {
		remote = info
		if remote.Model == "" {
			remote.Model = model
		}
	}

	volumeCreate, err := newCreateVolumeRemote(parameters, remote)
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
func (a *App) ModifyVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.VolumeModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}
	if parameters.Volume == "" {
		return nil, nil, errors.New("volume name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		volumeUpdate, err := updateVolumeValidation(parameters.VolumeUpdate)
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateVolume(ctx, volumeUpdate, parameters.Volume, parameters.SVM)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Volume updated successfully"},
			},
		}, nil, nil
	case "delete":
		volumeDelete := ontap.Volume{
			SVM:  ontap.NameAndUUID{Name: parameters.SVM},
			Name: parameters.Volume,
		}

		err = client.DeleteVolume(ctx, volumeDelete)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Volume deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

func updateVolumeValidation(in tool.VolumeUpdate) (ontap.Volume, error) {
	out := ontap.Volume{}

	hasUpdate := false
	if in.NewVolume != "" {
		out.Name = in.NewVolume
		hasUpdate = true
	}

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.Volume{}, err
		}
		out.Size = size
		hasUpdate = true
	}

	if in.State != "" {
		out.State = in.State
		hasUpdate = true
	}

	if in.JunctionPath != "" {
		out.Nas.Path = in.JunctionPath
		hasUpdate = true
	}

	if in.ExportPolicy != "" {
		out.Nas.ExportPolicy.Name = in.ExportPolicy
		hasUpdate = true
	}

	if in.FilesMaximum != nil {
		if *in.FilesMaximum <= 0 {
			return out, errors.New("files.maximum must be a positive integer")
		}
		out.Files.Maximum = in.FilesMaximum
		hasUpdate = true
	}

	if in.Autosize.Mode != "" {
		out.Autosize.Mode = in.Autosize.Mode
		hasUpdate = true
	}
	if in.Autosize.MaxSize != "" {
		out.Autosize.MaxSize = in.Autosize.MaxSize
		hasUpdate = true
	}
	if in.Autosize.MinSize != "" {
		out.Autosize.MinSize = in.Autosize.MinSize
		hasUpdate = true
	}
	if in.Autosize.GrowThreshold != "" {
		out.Autosize.GrowThreshold = in.Autosize.GrowThreshold
		hasUpdate = true
	}
	if in.Autosize.ShrinkThreshold != "" {
		out.Autosize.ShrinkThreshold = in.Autosize.ShrinkThreshold
		hasUpdate = true
	}

	switch {
	case in.QoS.RemovePolicy:
		out.QoS.Policy.Name = "none"
		hasUpdate = true
	case in.QoS.PolicyName != "":
		out.QoS.Policy.Name = in.QoS.PolicyName
		hasUpdate = true
	default:
		var err error
		if in.QoS.MaxIOPS != "" || in.QoS.MinIOPS != "" || in.QoS.MaxMBPS != "" || in.QoS.MinMBPS != "" {
			hasUpdate = true
		}
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

	if in.GuaranteeType != "" {
		out.Guarantee.Type = in.GuaranteeType
		hasUpdate = true
	}
	if in.SnapshotPolicyName != "" {
		out.SnapshotPolicy.Name = in.SnapshotPolicyName
		hasUpdate = true
	}
	if in.SnapshotReservePercent != nil {
		if *in.SnapshotReservePercent < 0 || *in.SnapshotReservePercent > 100 {
			return out, errors.New("snapshot_reserve_percent must be between 0 and 100")
		}
		out.Space.Snapshot.ReservePercent = in.SnapshotReservePercent
		hasUpdate = true
	}
	if in.Efficiency.Dedupe != "" {
		out.Efficiency.Dedupe = in.Efficiency.Dedupe
		hasUpdate = true
	}
	if in.Efficiency.CrossVolumeDedupe != "" {
		out.Efficiency.CrossVolumeDedupe = in.Efficiency.CrossVolumeDedupe
		hasUpdate = true
	}
	if in.Efficiency.Compression != "" {
		out.Efficiency.Compression = in.Efficiency.Compression
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided (e.g. new_volume_name, size, state, nas.path, nas.export_policy.name, files.maximum, autosize: mode/maximum/minimum/grow_threshold/shrink_threshold, qos.policy: name/remove_qos_policy/max_iops/min_iops/max_mbps/min_mbps, guarantee.type, snapshot_policy.name, space.snapshot.reserve_percent, efficiency)")
	}

	return out, nil
}

// newCreateVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API.
// model is ontap.CDOT, ontap.AFX, ontap.ASAr2, or empty (treated as CDOT).
func newCreateVolume(in tool.VolumeCreate, model string) (ontap.Volume, error) {
	return newCreateVolumeRemote(in, ontap.Remote{Model: model})
}

func newCreateVolumeRemote(in tool.VolumeCreate, remote ontap.Remote) (ontap.Volume, error) {
	out := ontap.Volume{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}

	model := remote.Model
	if model == ontap.ASAr2 {
		return out, errors.New("volume creation is not supported on ASAr2 clusters, use storage units instead")
	}

	style := strings.ToLower(strings.TrimSpace(in.Style))
	switch style {
	case "", "flexvol", "flexgroup":
	case "flexgroup_constituent":
		return out, errors.New("style flexgroup_constituent is not supported when creating a volume")
	default:
		return out, fmt.Errorf("unsupported volume style %q; use flexvol or flexgroup", in.Style)
	}

	isFlexGroup := style == "flexgroup"
	if !isFlexGroup && hasFlexGroupPlacementFields(in) {
		return out, errors.New("aggregate_names, constituents_per_aggregate, optimize_aggr_list, and granular_data are only valid when style is flexgroup")
	}
	if in.Aggregate != "" && len(in.AggregateNames) > 0 {
		return out, errors.New("cannot set both aggregate_name and aggregate_names")
	}

	if isFlexGroup {
		if model == ontap.AFX {
			return out, errors.New("AFX volume placement omits aggregates; do not set style, aggregate_names, constituents_per_aggregate, optimize_aggr_list, or granular_data")
		}
		if remote.IsSanOptimized {
			return out, errors.New("FlexGroup is not supported on All SAN Arrays")
		}
		if len(in.AggregateNames) == 0 {
			return out, errors.New("aggregate_names is required when style is flexgroup")
		}
		out.Style = "flexgroup"
		out.Aggregates = make([]ontap.NameAndUUID, 0, len(in.AggregateNames))
		for _, name := range in.AggregateNames {
			out.Aggregates = append(out.Aggregates, ontap.NameAndUUID{Name: name})
		}
		out.ConstituentsPerAggregate = in.ConstituentsPerAggregate
		if in.OptimizeAggrList != nil && *in.OptimizeAggrList {
			out.OptimizeAggregates = in.OptimizeAggrList
		}
		if err := applyGranularData(&out, in.GranularData); err != nil {
			return out, err
		}
	} else {
		switch model {
		case ontap.AFX:
			if in.Aggregate != "" {
				return out, errors.New("aggregate name must not be provided for AFX clusters")
			}
			if in.GuaranteeType != "" {
				return out, errors.New("space guarantee type must not be provided for AFX clusters")
			}
		default:
			if in.Aggregate == "" {
				return out, errors.New("aggregate name is required")
			}
			out.Aggregates = []ontap.NameAndUUID{
				{Name: in.Aggregate},
			}
		}
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Volume

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.Volume{}, err
		}
		out.Size = size
	}

	if in.Type != "" {
		out.Type = in.Type
	}
	if in.GuaranteeType != "" {
		out.Guarantee.Type = in.GuaranteeType
	}
	if in.SnapshotPolicyName != "" {
		out.SnapshotPolicy.Name = in.SnapshotPolicyName
	}
	if in.SnapshotReservePercent != nil {
		if *in.SnapshotReservePercent < 0 || *in.SnapshotReservePercent > 100 {
			return out, errors.New("snapshot_reserve_percent must be between 0 and 100")
		}
		out.Space.Snapshot.ReservePercent = in.SnapshotReservePercent
	}
	if in.Efficiency.Dedupe != "" {
		out.Efficiency.Dedupe = in.Efficiency.Dedupe
	}
	if in.Efficiency.CrossVolumeDedupe != "" {
		out.Efficiency.CrossVolumeDedupe = in.Efficiency.CrossVolumeDedupe
	}
	if in.Efficiency.Compression != "" {
		out.Efficiency.Compression = in.Efficiency.Compression
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

func hasFlexGroupPlacementFields(in tool.VolumeCreate) bool {
	return len(in.AggregateNames) > 0 || in.ConstituentsPerAggregate != nil || in.OptimizeAggrList != nil || strings.TrimSpace(in.GranularData) != ""
}

func applyGranularData(out *ontap.Volume, mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "disabled":
		return nil
	case "basic", "advanced":
		enabled := true
		out.GranularData = &enabled
		out.GranularDataMode = strings.ToLower(strings.TrimSpace(mode))
		return nil
	default:
		return fmt.Errorf("unsupported granular_data %q; use disabled, basic, or advanced", mode)
	}
}

// newUpdateVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateVolume(in tool.Volume) (ontap.Volume, error) {
	out := ontap.Volume{}
	hasUpdate := false
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.NewVolume != "" {
		out.Name = in.NewVolume
		hasUpdate = true
	}

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.Volume{}, err
		}
		out.Size = size
		hasUpdate = true
	}

	if in.State != "" {
		out.State = in.State
		hasUpdate = true
	}

	if in.JunctionPath != "" {
		out.Nas.Path = in.JunctionPath
		hasUpdate = true
	}

	if in.ExportPolicy != "" {
		out.Nas.ExportPolicy.Name = in.ExportPolicy
		hasUpdate = true
	}

	if in.Autosize.Mode != "" {
		out.Autosize.Mode = in.Autosize.Mode
		hasUpdate = true
	}
	if in.Autosize.MaxSize != "" {
		out.Autosize.MaxSize = in.Autosize.MaxSize
		hasUpdate = true
	}
	if in.Autosize.MinSize != "" {
		out.Autosize.MinSize = in.Autosize.MinSize
		hasUpdate = true
	}
	if in.Autosize.GrowThreshold != "" {
		out.Autosize.GrowThreshold = in.Autosize.GrowThreshold
		hasUpdate = true
	}
	if in.Autosize.ShrinkThreshold != "" {
		out.Autosize.ShrinkThreshold = in.Autosize.ShrinkThreshold
		hasUpdate = true
	}
	if in.Efficiency.Dedupe != "" {
		out.Efficiency.Dedupe = in.Efficiency.Dedupe
		hasUpdate = true
	}
	if in.Efficiency.CrossVolumeDedupe != "" {
		out.Efficiency.CrossVolumeDedupe = in.Efficiency.CrossVolumeDedupe
		hasUpdate = true
	}
	if in.Efficiency.Compression != "" {
		out.Efficiency.Compression = in.Efficiency.Compression
		hasUpdate = true
	}

	if in.GuaranteeType != "" {
		out.Guarantee.Type = in.GuaranteeType
		hasUpdate = true
	}
	if in.SnapshotPolicyName != "" {
		out.SnapshotPolicy.Name = in.SnapshotPolicyName
		hasUpdate = true
	}
	if in.SnapshotReservePercent != nil {
		if *in.SnapshotReservePercent < 0 || *in.SnapshotReservePercent > 100 {
			return out, errors.New("snapshot_reserve_percent must be between 0 and 100")
		}
		out.Space.Snapshot.ReservePercent = in.SnapshotReservePercent
		hasUpdate = true
	}

	switch {
	case in.QoS.RemovePolicy:
		out.QoS.Policy.Name = "none"
		hasUpdate = true
	case in.QoS.PolicyName != "":
		out.QoS.Policy.Name = in.QoS.PolicyName
		hasUpdate = true
	default:
		var err error
		if in.QoS.MaxIOPS != "" || in.QoS.MinIOPS != "" || in.QoS.MaxMBPS != "" || in.QoS.MinMBPS != "" {
			hasUpdate = true
		}
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

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided (e.g. new_volume_name, size, state, nas.path, nas.export_policy.name, autosize: mode/maximum/minimum/grow_threshold/shrink_threshold, qos.policy: name/remove_qos_policy/max_iops/min_iops/max_mbps/min_mbps, guarantee.type, snapshot_policy.name, space.snapshot.reserve_percent, efficiency)")
	}

	return out, nil
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
