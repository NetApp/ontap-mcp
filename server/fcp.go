package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateFCPService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.FCPService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	fcpServiceCreate, err := newCreateFCPService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateFCPService(ctx, fcpServiceCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "FCP service created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateFCPService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.FCPService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	fcpServiceUpdate, err := newUpdateFCPService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateFCPService(ctx, parameters.SVM, fcpServiceUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "FCP service updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteFCPService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.FCPService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := newDeleteFCPService(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteFCPService(ctx, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "FCP service deleted successfully"},
		},
	}, nil, nil
}

func newCreateFCPService(in tool.FCPService) (ontap.FCPService, error) {
	out := ontap.FCPService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	if in.Enabled != "" {
		out.Enabled = in.Enabled
	}
	return out, nil
}

func newUpdateFCPService(in tool.FCPService) (ontap.FCPService, error) {
	out := ontap.FCPService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Enabled != "" {
		out.Enabled = in.Enabled
	}
	return out, nil
}

func newDeleteFCPService(in tool.FCPService) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	return nil
}

func (a *App) CreateFCInterface(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.FCInterface) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	fcInterfaceCreate, err := newCreateFCInterface(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateFCInterface(ctx, fcInterfaceCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "FC interface created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateFCInterface(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.FCInterface) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	fcInterfaceUpdate, err := newUpdateFCInterface(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateFCInterface(ctx, parameters.SVM, parameters.Name, fcInterfaceUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "FC interface updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteFCInterface(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.FCInterface) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := newDeleteFCInterface(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteFCInterface(ctx, parameters.SVM, parameters.Name)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "FC interface deleted successfully"},
		},
	}, nil, nil
}

func newCreateFCInterface(in tool.FCInterface) (ontap.FCInterface, error) {
	out := ontap.FCInterface{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("FC interface name is required")
	}
	if in.DataProtocol == "" {
		return out, errors.New("data protocol is required")
	}
	if in.HomeNodeName == "" {
		return out, errors.New("home node name is required")
	}
	if in.HomePortName == "" {
		return out, errors.New("home port name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	out.DataProtocol = in.DataProtocol
	out.Location = ontap.FCInterfaceLocation{
		HomePort: ontap.FCInterfacePort{
			Name: in.HomePortName,
			Node: ontap.NameAndUUID{Name: in.HomeNodeName},
		},
	}
	if in.Enabled != "" {
		out.Enabled = in.Enabled
	}
	return out, nil
}

func newUpdateFCInterface(in tool.FCInterface) (ontap.FCInterface, error) {
	out := ontap.FCInterface{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("FC interface name is required")
	}
	if in.Enabled != "" {
		out.Enabled = in.Enabled
	}
	if (in.HomeNodeName == "" && in.HomePortName != "") || (in.HomeNodeName != "" && in.HomePortName == "") {
		return out, errors.New("both home_node_name and home_port_name must be provided together or both omitted")
	}
	if in.HomeNodeName != "" && in.HomePortName != "" {
		out.Location = ontap.FCInterfaceLocation{
			HomePort: ontap.FCInterfacePort{
				Name: in.HomePortName,
				Node: ontap.NameAndUUID{Name: in.HomeNodeName},
			},
		}
	}
	return out, nil
}

func newDeleteFCInterface(in tool.FCInterface) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	if in.Name == "" {
		return errors.New("FC interface name is required")
	}
	return nil
}
