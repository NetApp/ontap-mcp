package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirrorCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	rel, err := newCreateSnapMirror(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.CreateSnapMirror(ctx, rel); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SnapMirror relationship created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	rel, err := newUpdateSnapMirror(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.UpdateSnapMirror(ctx, parameters.DestinationSVM, parameters.DestinationVolume, rel); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SnapMirror relationship updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.DestinationSVM == "" {
		return nil, nil, errors.New("destination SVM name is required")
	}
	if parameters.DestinationVolume == "" {
		return nil, nil, errors.New("destination volume name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.DeleteSnapMirror(ctx, parameters.DestinationSVM, parameters.DestinationVolume); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SnapMirror relationship deleted successfully"},
		},
	}, nil, nil
}

func (a *App) InitializeSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	return a.InitializeSMUpdateSMTransfer(ctx, "SnapMirror relationship initialized successfully", parameters)
}

func (a *App) UpdateSnapMirrorTransfer(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	return a.InitializeSMUpdateSMTransfer(ctx, "SnapMirror transfer updated successfully", parameters)
}

func (a *App) InitializeSMUpdateSMTransfer(ctx context.Context, returnText string, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.DestinationSVM == "" {
		return nil, nil, errors.New("destination SVM name is required")
	}
	if parameters.DestinationVolume == "" {
		return nil, nil, errors.New("destination volume name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.InitializeSMUpdateSMTransfer(ctx, parameters.DestinationSVM, parameters.DestinationVolume); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: returnText},
		},
	}, nil, nil
}

func (a *App) BreakSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.DestinationSVM == "" {
		return nil, nil, errors.New("destination SVM name is required")
	}
	if parameters.DestinationVolume == "" {
		return nil, nil, errors.New("destination volume name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	rel := ontap.SnapMirrorRelationship{State: "broken_off"}

	if err := client.BreakSnapMirror(ctx, parameters.DestinationSVM, parameters.DestinationVolume, rel); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SnapMirror relationship broken successfully"},
		},
	}, nil, nil
}

func (a *App) ResyncSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.DestinationSVM == "" {
		return nil, nil, errors.New("destination SVM name is required")
	}
	if parameters.DestinationVolume == "" {
		return nil, nil, errors.New("destination volume name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	rel := ontap.SnapMirrorRelationship{State: "snapmirrored"}

	if err := client.ResyncSnapMirror(ctx, parameters.DestinationSVM, parameters.DestinationVolume, rel); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SnapMirror relationship resynced successfully"},
		},
	}, nil, nil
}

func newCreateSnapMirror(in tool.SnapMirrorCreate) (ontap.SnapMirrorRelationship, error) {
	if in.SourceSVM == "" {
		return ontap.SnapMirrorRelationship{}, errors.New("source SVM name is required")
	}
	if in.SourceVolume == "" {
		return ontap.SnapMirrorRelationship{}, errors.New("source volume name is required")
	}
	if in.DestinationSVM == "" {
		return ontap.SnapMirrorRelationship{}, errors.New("destination SVM name is required")
	}
	if in.DestinationVolume == "" {
		return ontap.SnapMirrorRelationship{}, errors.New("destination volume name is required")
	}
	if in.PolicyName == "" {
		return ontap.SnapMirrorRelationship{}, errors.New("policy name is required")
	}

	return ontap.SnapMirrorRelationship{
		Source:      ontap.SnapMirrorEndpoint{Path: fmt.Sprintf("%s:%s", in.SourceSVM, in.SourceVolume)},
		Destination: ontap.SnapMirrorEndpoint{Path: fmt.Sprintf("%s:%s", in.DestinationSVM, in.DestinationVolume)},
		Policy:      ontap.NameAndUUID{Name: in.PolicyName},
	}, nil
}

func newUpdateSnapMirror(in tool.SnapMirror) (ontap.SnapMirrorRelationship, error) {
	out := ontap.SnapMirrorRelationship{}
	if in.DestinationSVM == "" {
		return ontap.SnapMirrorRelationship{}, errors.New("destination SVM name is required")
	}
	if in.DestinationVolume == "" {
		return ontap.SnapMirrorRelationship{}, errors.New("destination volume name is required")
	}

	hasUpdate := false
	if in.PolicyName != "" {
		out.Policy = ontap.NameAndUUID{Name: in.PolicyName}
		hasUpdate = true
	}
	if in.TransferScheduleName != "" {
		out.TransferSchedule = ontap.NameAndUUID{Name: in.TransferScheduleName}
		hasUpdate = true
	}
	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: policy_name or transfer_schedule_name")
	}
	return out, nil
}
