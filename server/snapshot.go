package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateSnapshot(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Snapshot) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	snapshotCreate, err := newCreateSnapshot(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.CreateSnapshot(ctx, snapshotCreate, parameters.Volume, parameters.SVM); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Snapshot created successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteSnapshot(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Snapshot) (*mcp.CallToolResult, any, error) {
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
	if parameters.Name == "" {
		return nil, nil, errors.New("snapshot name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.DeleteSnapshot(ctx, parameters.Volume, parameters.SVM, parameters.Name); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Snapshot deleted successfully"},
		},
	}, nil, nil
}

func (a *App) RestoreSnapshot(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Snapshot) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	snapshotRestore, err := newRestoreSnapshot(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.RestoreSnapshot(ctx, parameters.Volume, parameters.SVM, snapshotRestore); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Snapshot restored successfully"},
		},
	}, nil, nil
}

func newCreateSnapshot(in tool.Snapshot) (ontap.Snapshot, error) {
	out := ontap.Snapshot{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Name == "" {
		return out, errors.New("snapshot name is required")
	}

	out.Name = in.Name
	return out, nil
}

func newRestoreSnapshot(in tool.Snapshot) (ontap.SnapshotRestore, error) {
	out := ontap.SnapshotRestore{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Name == "" {
		return out, errors.New("snapshot name is required")
	}

	out.RestoreTo.Snapshot.Name = in.Name
	return out, nil
}
