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

	if err := newDeleteIscsiService(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteIscsiService(ctx, parameters.SVM)

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

func (a *App) CreateNetworkIPInterface(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NetworkIPInterface) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	networkIPInterfaceCreate, err := newCreateNetworkIPInterface(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateNetworkIPInterface(ctx, networkIPInterfaceCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Network IP interface created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateNetworkIPInterface(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NetworkIPInterface) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	networkIPInterfaceUpdate, err := newUpdateNetworkIPInterface(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateNetworkIPInterface(ctx, parameters.Scope, parameters.Name, parameters.SVM, networkIPInterfaceUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Network IP interface updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteNetworkIPInterface(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NetworkIPInterface) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := newDeleteNetworkIPInterface(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteNetworkIPInterface(ctx, parameters.Scope, parameters.Name, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Network IP interface deleted successfully"

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

func newDeleteIscsiService(in tool.IscsiService) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	return nil
}

func newCreateNetworkIPInterface(in tool.NetworkIPInterface) (ontap.NetworkIPInterface, error) {
	out := ontap.NetworkIPInterface{}
	if in.Name == "" {
		return out, errors.New("network interface name is required")
	}
	out.Name = in.Name

	if in.Scope != "" {
		out.Scope = in.Scope
	}

	if in.IPAddress == "" && in.IPNetmask == "" && in.Subnet == "" {
		return out, errors.New("network IP address and IP netmask OR network subnet is required")
	}

	if in.Subnet != "" {
		out.Subnet.Name = in.Subnet
	} else {
		if in.IPAddress == "" || in.IPNetmask == "" {
			return out, errors.New("network IP address and IP netmask are required")
		}
		out.IP = ontap.IP{Address: in.IPAddress, Netmask: in.IPNetmask}
	}

	if in.HomeNode == "" && in.BroadcastDomain == "" {
		return out, errors.New("home node name OR broadcast domain is required")
	}
	if in.HomeNode != "" {
		out.Location.HomeNode.Name = in.HomeNode
	}
	if in.BroadcastDomain != "" {
		out.Location.BroadcastDomain.Name = in.BroadcastDomain
	}

	if in.SVM == "" && in.IPSpace == "" {
		return out, errors.New("SVM name OR IPSpace name is required")
	}

	if in.SVM != "" {
		out.SVM = ontap.NameAndUUID{Name: in.SVM}
	}
	if in.IPSpace != "" {
		out.IPSpace = ontap.NameAndUUID{Name: in.IPSpace}
	}

	return out, nil
}

func newUpdateNetworkIPInterface(in tool.NetworkIPInterface) (ontap.NetworkIPInterface, error) {
	out := ontap.NetworkIPInterface{}
	if in.Name == "" {
		return out, errors.New("network interface name is required")
	}
	if in.Scope == "" {
		return out, errors.New("scope is required")
	}
	if in.Scope == "svm" && in.SVM == "" {
		return out, errors.New("SVM name is required")
	}

	hasUpdates := false
	if in.AutoRevert != "" {
		out.Location.AutoRevert = in.AutoRevert
		hasUpdates = true
	}
	if in.ServicePolicy != "" {
		out.ServicePolicy.Name = in.ServicePolicy
		hasUpdates = true
	}
	if !hasUpdates {
		return out, errors.New("at least one supported update field must be provided; only auto_revert and service_policy are supported for update")
	}

	return out, nil
}

func newDeleteNetworkIPInterface(in tool.NetworkIPInterface) error {
	if in.Name == "" {
		return errors.New("network interface name is required")
	}
	if in.Scope == "" {
		return errors.New("scope is required")
	}
	if in.Scope == "svm" && in.SVM == "" {
		return errors.New("SVM name is required")
	}
	return nil
}
