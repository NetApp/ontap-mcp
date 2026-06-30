package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateCIFSShare(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSShareCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	cifsShareCreate, err := newCreateCIFSShare(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateCIFSShare(ctx, cifsShareCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "CIFS share created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateCIFSShare(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSShare) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	cifsShareUpdate, err := newUpdateCIFSShare(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateCIFSShare(ctx, parameters.SVM, parameters.Name, cifsShareUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "CIFS share updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteCIFSShare(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSShare) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	cifsShareDelete, err := newDeleteCIFSShare(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteCIFSShare(ctx, cifsShareDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "CIFS share deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newCreateCIFSShare validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateCIFSShare(in tool.CIFSShareCreate) (ontap.CIFSShare, error) {
	out := ontap.CIFSShare{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("cifs share name is required")
	}
	if in.Path == "" {
		return out, errors.New("cifs share path is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	out.Path = in.Path
	return out, nil
}

func (a *App) ModifyCIFSShare(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSShareModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}
	if parameters.Name == "" {
		return nil, nil, errors.New("CIFS share name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		cifsShareUpdate, err := updateCIFSShareValidation(parameters.CIFSShareUpdate)
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateCIFSShare(ctx, parameters.SVM, parameters.Name, cifsShareUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "CIFS share updated successfully"},
			},
		}, nil, nil
	case "delete":
		cifsShareDelete := ontap.CIFSShare{
			SVM:  ontap.NameAndUUID{Name: parameters.SVM},
			Name: parameters.Name,
		}

		err = client.DeleteCIFSShare(ctx, cifsShareDelete)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "CIFS share deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

func updateCIFSShareValidation(in tool.CIFSShareUpdate) (ontap.CIFSShare, error) {
	out := ontap.CIFSShare{}

	hasUpdate := false
	if in.Path != "" {
		out.Path = in.Path
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: path")
	}
	return out, nil
}

// newUpdateCIFSShare validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateCIFSShare(in tool.CIFSShare) (ontap.CIFSShare, error) {
	out := ontap.CIFSShare{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("cifs share name is required")
	}

	hasUpdate := false
	if in.Path != "" {
		out.Path = in.Path
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: path")
	}

	return out, nil
}

// newDeleteCIFSShare validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteCIFSShare(in tool.CIFSShare) (ontap.CIFSShare, error) {
	out := ontap.CIFSShare{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("cifs share name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	return out, nil
}
