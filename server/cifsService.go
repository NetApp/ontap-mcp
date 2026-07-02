package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateCIFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSServiceCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	cifsService, err := newCreateCIFSService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.CreateCIFSService(ctx, cifsService)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "CIFS service created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateCIFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	cifsService, err := newUpdateCIFSService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.UpdateCIFSService(ctx, parameters.SVM, cifsService)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "CIFS service updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteCIFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	if (parameters.ADUser == "") != (parameters.ADPassword == "") {
		return nil, nil, errors.New("both ad_user and ad_password must be provided together, or neither")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.DeleteCIFSService(ctx, parameters.SVM, parameters.ADUser, parameters.ADPassword)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "CIFS service deleted successfully"},
		},
	}, nil, nil
}

func (a *App) ModifyCIFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSServiceModify) (*mcp.CallToolResult, any, error) {
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

	switch parameters.Operation {
	case "update":
		cifsService, err := updateCIFSServiceValidation(parameters.CIFSServiceUpdate)
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateCIFSService(ctx, parameters.SVM, cifsService)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "CIFS service updated successfully"},
			},
		}, nil, nil
	case "delete":
		if (parameters.ADUser == "") != (parameters.ADPassword == "") {
			return nil, nil, errors.New("both ad_user and ad_password must be provided together, or neither")
		}

		err = client.DeleteCIFSService(ctx, parameters.SVM, parameters.ADUser, parameters.ADPassword)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "CIFS service deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

func updateCIFSServiceValidation(in tool.CIFSServiceUpdate) (ontap.CIFSServiceBody, error) {
	out := ontap.CIFSServiceBody{}

	hasUpdate := false
	if in.Name != "" {
		out.Name = in.Name
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: cifs_server_name")
	}

	return out, nil
}

func newCreateCIFSService(in tool.CIFSServiceCreate) (ontap.CIFSServiceBody, error) {
	out := ontap.CIFSServiceBody{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("CIFS server name is required")
	}
	if in.ADDomain == "" {
		return out, errors.New("AD domain FQDN is required")
	}
	if in.ADUser == "" {
		return out, errors.New("AD admin username is required")
	}
	if in.ADPassword == "" {
		return out, errors.New("AD admin password is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	out.ADDomain.FQDN = in.ADDomain
	out.ADDomain.User = in.ADUser
	out.ADDomain.Password = in.ADPassword
	if in.ADOu != "" {
		out.ADDomain.OrganizationalUnit = in.ADOu
	}

	return out, nil
}

func newUpdateCIFSService(in tool.CIFSService) (ontap.CIFSServiceBody, error) {
	out := ontap.CIFSServiceBody{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}

	hasUpdate := false
	if in.Name != "" {
		out.Name = in.Name
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: cifs_server_name")
	}

	return out, nil
}
