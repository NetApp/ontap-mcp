package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateSVM(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVMCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	svmCreate, err := newCreateSVM(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.CreateSVM(ctx, svmCreate)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateSVM(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVM) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	svmUpdate, err := newUpdateSVM(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.UpdateSVM(ctx, svmUpdate, parameters.Name)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteSVM(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVM) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.Name == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.DeleteSVM(ctx, parameters.Name)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM deleted successfully"},
		},
	}, nil, nil
}

func newCreateSVM(in tool.SVMCreate) (ontap.SVMCreate, error) {
	out := ontap.SVMCreate{}
	if in.Name == "" {
		return out, errors.New("SVM name is required")
	}
	out.Name = in.Name
	return out, nil
}

func newUpdateSVM(in tool.SVM) (ontap.SVM, error) {
	out := ontap.SVM{}

	if in.Name == "" {
		return out, errors.New("SVM name is required")
	}

	hasUpdate := false
	if in.NewName != "" {
		out.Name = in.NewName
		hasUpdate = true
	}
	if in.Comment != "" {
		out.Comment = in.Comment
		hasUpdate = true
	}
	if in.State != "" {
		out.State = in.State
		hasUpdate = true
	}
	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name, comment, or state")
	}

	return out, nil
}
