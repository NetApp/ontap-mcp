package server

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateClusterPeer(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.ClusterPeer) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.SourceCluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on source cluster %s, please try again", parameters.SourceCluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.SourceCluster)
	if !a.locks.TryLock(parameters.DestinationCluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on destination cluster %s, please try again", parameters.DestinationCluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.DestinationCluster)

	sourceClient, err := a.getClient(parameters.SourceCluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	destinationClient, err := a.getClient(parameters.DestinationCluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = sourceClient.CreateClusterPeer(ctx, destinationClient, parameters.SourceCluster, parameters.DestinationCluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "cluster peer relationship created successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteClusterPeer(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.ClusterPeer) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.SourceCluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on source cluster %s, please try again", parameters.SourceCluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.SourceCluster)
	if !a.locks.TryLock(parameters.DestinationCluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on destination cluster %s, please try again", parameters.DestinationCluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.DestinationCluster)

	sourceClient, err := a.getClient(parameters.SourceCluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	destinationClient, err := a.getClient(parameters.DestinationCluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = sourceClient.DeleteClusterPeer(ctx, destinationClient)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "cluster peer relationship deleted successfully"},
		},
	}, nil, nil
}
