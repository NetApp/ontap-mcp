package server

import (
	"context"
	"errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"strings"
)

func (a *App) ListCIFSShare(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSShare) (*mcp.CallToolResult, any, error) {
	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	cifsShares, err := client.GetCIFSShare(ctx)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(cifsShares, ",")},
		},
	}, nil, nil
}

func (a *App) CreateCIFSShare(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.CIFSShare) (*mcp.CallToolResult, any, error) {
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
func newCreateCIFSShare(in tool.CIFSShare) (ontap.CIFSShare, error) {
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
	if in.Path != "" {
		out.Path = in.Path
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
