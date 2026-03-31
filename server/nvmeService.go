package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateNVMeService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	nvmeServiceCreate, err := newCreateNVMeService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateNVMeService(ctx, nvmeServiceCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Service created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateNVMeService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	nvmeServiceUpdate, err := newUpdateNVMeService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateNVMeService(ctx, parameters.SVM, nvmeServiceUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Service updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteNVMeService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := newDeleteNVMeService(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteNVMeService(ctx, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Service deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func newCreateNVMeService(in tool.NVMeService) (ontap.NVMeService, error) {
	out := ontap.NVMeService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Enabled = in.Enabled
	return out, nil
}

func newUpdateNVMeService(in tool.NVMeService) (ontap.NVMeService, error) {
	out := ontap.NVMeService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	out.Enabled = in.Enabled
	return out, nil
}

func newDeleteNVMeService(in tool.NVMeService) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	return nil
}
