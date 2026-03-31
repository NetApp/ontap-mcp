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

func (a *App) CreateNVMeSubsystem(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeSubsystem) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	nvmeSubsystemCreate, err := newCreateNVMeSubsystem(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateNVMeSubsystem(ctx, nvmeSubsystemCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Subsystem created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateNVMeSubsystem(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeSubsystem) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	nvmeSubsystemUpdate, err := newUpdateNVMeSubsystem(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateNVMeSubsystem(ctx, parameters.SVM, parameters.Name, parameters.OSType, nvmeSubsystemUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Subsystem updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteNVMeSubsystem(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeSubsystem) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := newDeleteNVMeSubsystem(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteNVMeSubsystem(ctx, parameters.SVM, parameters.Name, parameters.OSType)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Subsystem deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func newCreateNVMeSubsystem(in tool.NVMeSubsystem) (ontap.NVMeSubsystem, error) {
	out := ontap.NVMeSubsystem{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("NVMe subsystem name is required")
	}
	if in.OSType == "" {
		return out, errors.New("OS type is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	out.OSType = in.OSType

	for _, nqn := range in.HostNQNs {
		out.Hosts = append(out.Hosts, ontap.Hosts{NQN: nqn})
	}

	return out, nil
}

func newUpdateNVMeSubsystem(in tool.NVMeSubsystem) (ontap.NVMeSubsystem, error) {
	out := ontap.NVMeSubsystem{}

	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("NVMe subsystem name is required")
	}
	if in.OSType == "" {
		return out, errors.New("OS type is required")
	}

	if in.Comment != "" {
		out.Comment = in.Comment
	}
	return out, nil
}

func newDeleteNVMeSubsystem(in tool.NVMeSubsystem) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	if in.Name == "" {
		return errors.New("NVMe subsystem name is required")
	}
	if in.OSType == "" {
		return errors.New("OS type is required")
	}
	return nil
}

func (a *App) AddNVMeSubsystemHost(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeSubsystemHost) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	nvmeSubsystemHostAdd, err := newAddNVMeSubsystemHost(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.AddNVMeSubsystemHost(ctx, parameters.SVM, parameters.Name, parameters.OSType, nvmeSubsystemHostAdd)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Subsystem Host added successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) RemoveNVMeSubsystemHost(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NVMeSubsystemHost) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := newRemoveNVMeSubsystemHost(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.RemoveNVMeSubsystemHost(ctx, parameters.SVM, parameters.Name, parameters.OSType, parameters.NQN)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NVMe Subsystem Host removed successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func newAddNVMeSubsystemHost(in tool.NVMeSubsystemHost) (ontap.NVMeSubsystemHost, error) {
	out := ontap.NVMeSubsystemHost{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("NVMe subsystem name is required")
	}
	if in.OSType == "" {
		return out, errors.New("OS type is required")
	}

	if in.NQN == "" && len(in.Records) == 0 {
		return out, errors.New("NVMe subsystem host NQN OR Array of NQNs are required")
	}
	if in.NQN != "" {
		out.NQN = in.NQN
	}

	for _, nqn := range in.Records {
		out.Records = append(out.Records, ontap.Hosts{NQN: nqn})
	}
	return out, nil
}

func newRemoveNVMeSubsystemHost(in tool.NVMeSubsystemHost) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	if in.Name == "" {
		return errors.New("NVMe subsystem name is required")
	}
	if in.OSType == "" {
		return errors.New("OS type is required")
	}
	if in.NQN == "" {
		return errors.New("NQN is required")
	}
	return nil
}
