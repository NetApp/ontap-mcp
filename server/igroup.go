package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateIGroup(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IGroupCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	igroupCreate, err := newCreateIGroup(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateIGroup(ctx, igroupCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "igroup created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateIGroup(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IGroup) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	igroupUpdate, err := newUpdateIGroup(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateIGroup(ctx, igroupUpdate, parameters.Name, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "igroup updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteIGroup(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IGroup) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	igroupDelete, err := newDeleteIGroup(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteIGroup(ctx, igroupDelete, parameters.AllowDeleteWhileMapped)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "igroup deleted successfully"},
		},
	}, nil, nil
}

func (a *App) AddIGroupInitiator(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IGroupInitiator) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	initiatorAdd, err := addIGroupInitiator(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.AddIGroupInitiator(ctx, parameters.IGroupName, parameters.SVM, initiatorAdd)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "initiator added to igroup successfully"},
		},
	}, nil, nil
}

func (a *App) RemoveIGroupInitiator(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.IGroupInitiator) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	initiatorRemove, err := removeIGroupInitiator(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.RemoveIGroupInitiator(ctx, parameters.IGroupName, parameters.SVM, initiatorRemove, parameters.AllowDeleteWhileMapped)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "initiator removed from igroup successfully"},
		},
	}, nil, nil
}

func newCreateIGroup(in tool.IGroupCreate) (ontap.IGroup, error) {
	out := ontap.IGroup{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("igroup name is required")
	}
	if in.OSType == "" {
		return out, errors.New("OS type is required")
	}
	if in.Protocol == "" {
		return out, errors.New("protocol is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	out.OSType = in.OSType
	out.Protocol = in.Protocol
	if in.Comment != "" {
		out.Comment = in.Comment
	}
	return out, nil
}

func newUpdateIGroup(in tool.IGroup) (ontap.IGroup, error) {
	out := ontap.IGroup{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("igroup name is required")
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
	if in.OSType != "" {
		out.OSType = in.OSType
		hasUpdate = true
	}
	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name, comment, or os_type")
	}
	return out, nil
}

func newDeleteIGroup(in tool.IGroup) (ontap.IGroup, error) {
	out := ontap.IGroup{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("igroup name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	return out, nil
}

func addIGroupInitiator(in tool.IGroupInitiator) (ontap.IGroupInitiator, error) {
	out := ontap.IGroupInitiator{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.IGroupName == "" {
		return out, errors.New("igroup name is required")
	}

	if in.Comment != "" {
		out.Comment = in.Comment
	}

	if in.InitiatorName == "" && len(in.Records) == 0 {
		return out, errors.New("either initiator name OR one or more initiator names (records) must be provided")
	}

	// Enforce mutual exclusivity: cannot specify both a single initiator name and an array
	if in.InitiatorName != "" && len(in.Records) > 0 {
		return out, errors.New("specify either a single initiator name or an array of initiator names, but not both")
	}

	if in.InitiatorName != "" {
		out.Name = in.InitiatorName
		return out, nil
	}

	for _, iName := range in.Records {
		if iName == "" {
			return out, errors.New("all initiator names in the array must be non-empty")
		}
		out.Records = append(out.Records, ontap.InitiatorName{Name: iName})
	}

	return out, nil
}

func removeIGroupInitiator(in tool.IGroupInitiator) (ontap.IGroupInitiator, error) {
	out := ontap.IGroupInitiator{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.IGroupName == "" {
		return out, errors.New("igroup name is required")
	}
	if in.InitiatorName == "" {
		return out, errors.New("initiator name is required")
	}
	out.Name = in.InitiatorName
	return out, nil
}
