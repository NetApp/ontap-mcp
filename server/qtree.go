package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateQtree(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Qtree) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	qtreeCreate, err := newCreateQtree(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateQtree(ctx, qtreeCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Qtree created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateQtree(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Qtree) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	qtreeUpdate, err := newUpdateQtree(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateQtree(ctx, parameters.SVM, parameters.Volume, parameters.Name, qtreeUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Qtree updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteQtree(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Qtree) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	qtreeDelete, err := newDeleteQtree(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteQtree(ctx, qtreeDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Qtree deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func newCreateQtree(in tool.Qtree) (ontap.Qtree, error) {
	out := ontap.Qtree{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Name == "" {
		return out, errors.New("qtree name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Volume = ontap.NameAndUUID{Name: in.Volume}
	out.Name = in.Name
	return out, nil
}

func newUpdateQtree(in tool.Qtree) (ontap.Qtree, error) {
	out := ontap.Qtree{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Name == "" {
		return out, errors.New("qtree name is required")
	}
	out.Name = in.Name
	if in.NewName != "" {
		out.Name = in.NewName
	}
	return out, nil
}

func newDeleteQtree(in tool.Qtree) (ontap.Qtree, error) {
	out := ontap.Qtree{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Name == "" {
		return out, errors.New("qtree name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Volume = ontap.NameAndUUID{Name: in.Volume}
	out.Name = in.Name
	return out, nil
}
