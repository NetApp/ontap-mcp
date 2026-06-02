package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateNFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	nfsService, err := newCreateNFSService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.CreateNFSService(ctx, nfsService)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "NFS service created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateNFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	nfsService, err := newUpdateNFSService(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.UpdateNFSService(ctx, parameters.SVM, nfsService)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "NFS service updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteNFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSService) (*mcp.CallToolResult, any, error) {
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

	err = client.DeleteNFSService(ctx, parameters.SVM)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "NFS service deleted successfully"},
		},
	}, nil, nil
}

func newCreateNFSService(in tool.NFSService) (ontap.NFSService, error) {
	out := ontap.NFSService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}

	enabled := true
	if in.Enabled != "" {
		b, err := parseBool(in.Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for enabled: %s", in.Enabled)
		}
		enabled = b
	}
	out.Enabled = &enabled

	if in.V3Enabled != "" {
		b, err := parseBool(in.V3Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for v3_enabled: %s", in.V3Enabled)
		}
		out.Protocol.V3Enabled = &b
	}
	if in.V40Enabled != "" {
		b, err := parseBool(in.V40Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for v40_enabled: %s", in.V40Enabled)
		}
		out.Protocol.V40Enabled = &b
	}
	if in.V41Enabled != "" {
		b, err := parseBool(in.V41Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for v41_enabled: %s", in.V41Enabled)
		}
		out.Protocol.V41Enabled = &b
	}

	return out, nil
}

func newUpdateNFSService(in tool.NFSService) (ontap.NFSService, error) {
	out := ontap.NFSService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}

	hasUpdate := false
	if in.Enabled != "" {
		b, err := parseBool(in.Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for enabled: %s", in.Enabled)
		}
		out.Enabled = &b
		hasUpdate = true
	}
	if in.V3Enabled != "" {
		b, err := parseBool(in.V3Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for v3_enabled: %s", in.V3Enabled)
		}
		out.Protocol.V3Enabled = &b
		hasUpdate = true
	}
	if in.V40Enabled != "" {
		b, err := parseBool(in.V40Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for v40_enabled: %s", in.V40Enabled)
		}
		out.Protocol.V40Enabled = &b
		hasUpdate = true
	}
	if in.V41Enabled != "" {
		b, err := parseBool(in.V41Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid value for v41_enabled: %s", in.V41Enabled)
		}
		out.Protocol.V41Enabled = &b
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: enabled, v3_enabled, v40_enabled, or v41_enabled")
	}

	return out, nil
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true", "True", "TRUE", "1":
		return true, nil
	case "false", "False", "FALSE", "0":
		return false, nil
	default:
		return false, fmt.Errorf("cannot parse %q as boolean", s)
	}
}

