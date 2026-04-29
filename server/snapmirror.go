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
	rel, err := newUpdateSnapMirror(parameters)
	if err != nil {
		return nil, nil, err
	}
	return a.updateSnapMirrorState(ctx, parameters, rel, "SnapMirror relationship updated successfully")
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
	if err := validateDestination(parameters); err != nil {
		return nil, nil, err
	}

	rel := ontap.SnapMirrorRelationship{State: "snapmirrored"}
	return a.updateSnapMirrorState(ctx, parameters, rel, "SnapMirror relationship initialized successfully")
}

func (a *App) UpdateSnapMirrorTransfer(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
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

	if err := client.UpdateSnapMirrorTransfer(ctx, parameters.DestinationSVM, parameters.DestinationVolume); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SnapMirror transfer updated successfully"},
		},
	}, nil, nil
}

func (a *App) BreakSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	if err := validateDestination(parameters); err != nil {
		return nil, nil, err
	}

	rel := ontap.SnapMirrorRelationship{State: "broken_off"}
	return a.updateSnapMirrorState(ctx, parameters, rel, "SnapMirror relationship broken successfully")
}

func (a *App) ResyncSnapMirror(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapMirror) (*mcp.CallToolResult, any, error) {
	if err := validateDestination(parameters); err != nil {
		return nil, nil, err
	}

	rel := ontap.SnapMirrorRelationship{State: "snapmirrored"}
	return a.updateSnapMirrorState(ctx, parameters, rel, "SnapMirror relationship resynced successfully")
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
	if err := validateDestination(in); err != nil {
		return out, err
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
	if in.State != "" {
		out.State = in.State
		hasUpdate = true
	}
	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: policy_name, transfer_schedule_name or state")
	}
	return out, nil
}

func validateDestination(in tool.SnapMirror) error {
	if in.DestinationSVM == "" {
		return errors.New("destination SVM name is required")
	}
	if in.DestinationVolume == "" {
		return errors.New("destination volume name is required")
	}
	return nil
}

func (a *App) updateSnapMirrorState(ctx context.Context, parameters tool.SnapMirror, rel ontap.SnapMirrorRelationship, returnText string) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.UpdateSnapMirror(ctx, parameters.DestinationSVM, parameters.DestinationVolume, rel); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: returnText},
		},
	}, nil, nil
}
