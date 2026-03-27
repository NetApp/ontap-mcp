package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateIscsiService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IscsiService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	iscsiServiceCreate, err := newCreateIscsiService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateIscsiService(ctx, iscsiServiceCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "iSCSI Service created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateIscsiService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IscsiService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	iscsiServiceUpdate, err := newUpdateIscsiService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateIscsiService(ctx, parameters.SVM, iscsiServiceUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "iSCSI Service updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteIscsiService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IscsiService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	iscsiServiceDelete, err := newDeleteIscsiService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteIscsiService(ctx, iscsiServiceDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "iSCSI Service deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func newCreateIscsiService(in tool.IscsiService) (ontap.IscsiService, error) {
	out := ontap.IscsiService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.TargetAlias != "" {
		out.Target.Alias = in.TargetAlias
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Enabled = in.Enabled
	return out, nil
}

func newUpdateIscsiService(in tool.IscsiService) (ontap.IscsiService, error) {
	out := ontap.IscsiService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	out.Enabled = in.Enabled
	return out, nil
}

func newDeleteIscsiService(in tool.IscsiService) (ontap.IscsiService, error) {
	out := ontap.IscsiService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	return out, nil
}
