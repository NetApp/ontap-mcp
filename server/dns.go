package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateDNS(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.DNSService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	dns, err := newCreateDNS(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, nil
	}

	err = client.CreateDNS(ctx, dns)
	if err != nil {
		return errorResult(err), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "DNS configuration created successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteDNS(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.DNSService) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, nil
	}

	err = client.DeleteDNS(ctx, parameters.SVM)
	if err != nil {
		return errorResult(err), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "DNS configuration deleted successfully"},
		},
	}, nil, nil
}

func newCreateDNS(in tool.DNSService) (ontap.DNSConfig, error) {
	out := ontap.DNSConfig{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if len(in.Domains) == 0 {
		return out, errors.New("at least one DNS domain is required")
	}
	if len(in.Servers) == 0 {
		return out, errors.New("at least one DNS server is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Domains = in.Domains
	out.Servers = in.Servers
	out.SkipConfigValidation = in.SkipConfigValidation
	return out, nil
}
