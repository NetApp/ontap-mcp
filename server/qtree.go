package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateQtree(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QtreeCreate) (*mcp.CallToolResult, any, error) {
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

func (a *App) ModifyQtree(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QtreeModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := validateQtree(parameters.SVM, parameters.Volume, parameters.Name); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		qtreeUpdate, err := newUpdateQtree(tool.Qtree{
			SVM:     parameters.SVM,
			Volume:  parameters.Volume,
			Name:    parameters.Name,
			NewName: parameters.QtreeUpdate.NewName,
		})
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateQtree(ctx, parameters.SVM, parameters.Volume, parameters.Name, qtreeUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Qtree updated successfully"}}}, nil, nil
	case "delete":
		qtreeDelete, err := newDeleteQtree(tool.Qtree{
			SVM:    parameters.SVM,
			Volume: parameters.Volume,
			Name:   parameters.Name,
		})
		if err != nil {
			return nil, nil, err
		}

		err = client.DeleteQtree(ctx, qtreeDelete)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Qtree deleted successfully"}}}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

func newCreateQtree(in tool.QtreeCreate) (ontap.Qtree, error) {
	out := ontap.Qtree{}
	if err := validateQtree(in.SVM, in.Volume, in.Name); err != nil {
		return out, err
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Volume = ontap.NameAndUUID{Name: in.Volume}
	out.Name = in.Name
	return out, nil
}

func newUpdateQtree(in tool.Qtree) (ontap.Qtree, error) {
	out := ontap.Qtree{}
	hasUpdate := false

	if err := validateQtree(in.SVM, in.Volume, in.Name); err != nil {
		return out, err
	}

	out.Name = in.Name
	if in.NewName != "" {
		out.Name = in.NewName
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name")
	}

	return out, nil
}

func newDeleteQtree(in tool.Qtree) (ontap.Qtree, error) {
	out := ontap.Qtree{}

	if err := validateQtree(in.SVM, in.Volume, in.Name); err != nil {
		return out, err
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Volume = ontap.NameAndUUID{Name: in.Volume}
	out.Name = in.Name
	return out, nil
}

func validateQtree(svm, volume, name string) error {
	if svm == "" {
		return errors.New("SVM name is required")
	}
	if volume == "" {
		return errors.New("volume name is required")
	}
	if name == "" {
		return errors.New("qtree name is required")
	}
	return nil
}
