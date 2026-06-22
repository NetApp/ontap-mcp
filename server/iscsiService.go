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

func (a *App) ModifyIscsiService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IscsiServiceModify) (*mcp.CallToolResult, any, error) {
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
		iscsiServiceUpdate, err := newUpdateIscsiService(tool.IscsiService{SVM: parameters.SVM, Enabled: parameters.IscsiServiceUpdate.Enabled})
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateIscsiService(ctx, parameters.SVM, iscsiServiceUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "iSCSI Service updated successfully"}}}, nil, nil
	case "delete":
		err = client.DeleteIscsiService(ctx, parameters.SVM)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "iSCSI Service deleted successfully"}}}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
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

	if err := validateNwInterface(parameters.Name, parameters.Scope, parameters.SVM); err != nil {
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

func (a *App) ModifyNetworkIPInterface(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NetworkIPInterfaceModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := validateNwInterface(parameters.Name, parameters.Scope, parameters.SVM); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		networkIPInterfaceUpdate, err := newUpdateNetworkIPInterface(tool.NetworkIPInterface{
			Name:          parameters.Name,
			Scope:         parameters.Scope,
			SVM:           parameters.SVM,
			AutoRevert:    parameters.NetworkIPInterfaceUpdate.AutoRevert,
			ServicePolicy: parameters.NetworkIPInterfaceUpdate.ServicePolicy,
		})
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateNetworkIPInterface(ctx, parameters.Scope, parameters.Name, parameters.SVM, networkIPInterfaceUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Network IP interface updated successfully"}}}, nil, nil
	case "delete":
		err = client.DeleteNetworkIPInterface(ctx, parameters.Scope, parameters.Name, parameters.SVM)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Network IP interface deleted successfully"}}}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
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
	if in.ServicePolicy != "" {
		out.ServicePolicy.Name = in.ServicePolicy
	}

	return out, nil
}

func newUpdateNetworkIPInterface(in tool.NetworkIPInterface) (ontap.NetworkIPInterface, error) {
	out := ontap.NetworkIPInterface{}

	if err := validateNwInterface(in.Name, in.Scope, in.SVM); err != nil {
		return out, err
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

func validateNwInterface(name, scope, svm string) error {
	if name == "" {
		return errors.New("network interface name is required")
	}
	if scope == "" {
		return errors.New("scope is required")
	}
	if scope == "svm" && svm == "" {
		return errors.New("SVM name is required")
	}
	return nil
}
