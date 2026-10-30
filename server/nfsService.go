package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

func (a *App) ModifyNFSService(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSServiceModify) (*mcp.CallToolResult, any, error) {
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
		nfsService, err := updateNFSServiceValidation(parameters.NFSServiceUpdate)
		if err != nil {
			return nil, nil, err
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
	case "delete":
		err = client.DeleteNFSService(ctx, parameters.SVM)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "NFS service deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

const nfsServiceUpdateRequired = "at least one updatable field must be provided: enabled, v3_enabled, v40_enabled, v41_enabled, v41_pnfs_enabled, v41_trunking_enabled, rdma_enabled, v42_enabled, or v4_id_domain"

type nfsServiceFields struct {
	enabled    string
	v3         string
	v40        string
	v41        string
	pnfs       string
	trunking   string
	rdma       string
	v42        string
	v4IDDomain string
}

func nfsFieldsFromService(in tool.NFSService) nfsServiceFields {
	return nfsServiceFields{
		enabled:    in.Enabled,
		v3:         in.V3Enabled,
		v40:        in.V40Enabled,
		v41:        in.V41Enabled,
		pnfs:       in.V41PNFSEnabled,
		trunking:   in.V41TrunkingEnabled,
		rdma:       in.RDMAEnabled,
		v42:        in.V42Enabled,
		v4IDDomain: in.V4IDDomain,
	}
}

func nfsFieldsFromUpdate(in tool.NFSServiceUpdate) nfsServiceFields {
	return nfsServiceFields{
		enabled:    in.Enabled,
		v3:         in.V3Enabled,
		v40:        in.V40Enabled,
		v41:        in.V41Enabled,
		pnfs:       in.V41PNFSEnabled,
		trunking:   in.V41TrunkingEnabled,
		rdma:       in.RDMAEnabled,
		v42:        in.V42Enabled,
		v4IDDomain: in.V4IDDomain,
	}
}

// parseOptionalBool parses a true/false tool argument. An empty value is left unset.
func parseOptionalBool(name, value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value for %s %q (must be true/false): %w", name, value, err)
	}
	return &b, nil
}

func applyNFSServiceFields(out *ontap.NFSService, in nfsServiceFields) (bool, error) {
	hasUpdate := false
	take := func(name, value string) (*bool, error) {
		parsed, err := parseOptionalBool(name, value)
		if err != nil {
			return nil, err
		}
		if parsed != nil {
			hasUpdate = true
		}
		return parsed, nil
	}

	enabled, err := take("enabled", in.enabled)
	if err != nil {
		return false, err
	}
	out.Enabled = enabled

	v3, err := take("v3_enabled", in.v3)
	if err != nil {
		return false, err
	}
	out.Protocol.V3Enabled = v3

	v40, err := take("v40_enabled", in.v40)
	if err != nil {
		return false, err
	}
	out.Protocol.V40Enabled = v40

	v41, err := take("v41_enabled", in.v41)
	if err != nil {
		return false, err
	}
	out.Protocol.V41Enabled = v41

	pnfs, err := take("v41_pnfs_enabled", in.pnfs)
	if err != nil {
		return false, err
	}
	out.Protocol.V41Features.PNFSEnabled = pnfs

	trunking, err := take("v41_trunking_enabled", in.trunking)
	if err != nil {
		return false, err
	}
	out.Protocol.V41Features.TrunkingEnabled = trunking

	rdma, err := take("rdma_enabled", in.rdma)
	if err != nil {
		return false, err
	}
	out.Transport.RDMAEnabled = rdma

	v42, err := take("v42_enabled", in.v42)
	if err != nil {
		return false, err
	}
	out.Protocol.V42Enabled = v42

	if d := strings.TrimSpace(in.v4IDDomain); d != "" {
		out.Protocol.V4IDDomain = d
		hasUpdate = true
	}

	if v41 != nil && !*v41 {
		if pnfs != nil && *pnfs {
			return false, errors.New("v41_pnfs_enabled cannot be true when v41_enabled is false")
		}
		if trunking != nil && *trunking {
			return false, errors.New("v41_trunking_enabled cannot be true when v41_enabled is false")
		}
	}

	return hasUpdate, nil
}

func updateNFSServiceValidation(in tool.NFSServiceUpdate) (ontap.NFSService, error) {
	out := ontap.NFSService{}
	hasUpdate, err := applyNFSServiceFields(&out, nfsFieldsFromUpdate(in))
	if err != nil {
		return out, err
	}
	if !hasUpdate {
		return out, errors.New(nfsServiceUpdateRequired)
	}
	return out, nil
}

func newCreateNFSService(in tool.NFSService) (ontap.NFSService, error) {
	out := ontap.NFSService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}

	if _, err := applyNFSServiceFields(&out, nfsFieldsFromService(in)); err != nil {
		return out, err
	}
	if out.Enabled == nil {
		enabled := true
		out.Enabled = &enabled
	}
	return out, nil
}

func newUpdateNFSService(in tool.NFSService) (ontap.NFSService, error) {
	out := ontap.NFSService{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}

	hasUpdate, err := applyNFSServiceFields(&out, nfsFieldsFromService(in))
	if err != nil {
		return out, err
	}
	if !hasUpdate {
		return out, errors.New(nfsServiceUpdateRequired)
	}
	return out, nil
}
