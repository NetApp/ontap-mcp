package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) SVMOperation(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVMOperation) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if parameters.Name == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	switch parameters.Operation {
	case "create":
		svmCreate := ontap.SVMCreate{Name: parameters.Name}
		err = client.CreateSVM(ctx, svmCreate)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "SVM created successfully"},
			},
		}, nil, nil
	case "update":
		svmUpdate, err := updateSVMValidation(parameters.SVMUpdate)
		if err != nil {
			return nil, nil, err
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
	case "delete":
		err = client.DeleteSVM(ctx, parameters.Name)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "SVM deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported type_operation %q; supported values: create, update, delete", parameters.Operation)), nil, nil
	}
}

func updateSVMValidation(in tool.SVMUpdate) (ontap.SVM, error) {
	out := ontap.SVM{}

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

func (a *App) DeleteSVMPeer(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVMPeer) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.DeleteSVMPeer(ctx, parameters.SVM)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM peer deleted successfully"},
		},
	}, nil, nil
}
