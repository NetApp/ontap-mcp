package server

import (
	"context"
	"errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"strings"
)

func (a *App) ListNFSExportPolicies(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSExportPolicy) (*mcp.CallToolResult, any, error) {
	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	nfsExportPolicies, err := client.GetNFSExportPolicy(ctx)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(nfsExportPolicies, ",")},
		},
	}, nil, nil
}

func (a *App) CreateNFSExportPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSExportPolicy) (*mcp.CallToolResult, any, error) {
	nfsExportPolicyCreate, err := newCreateNFSExportPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateNFSExportPolicy(ctx, nfsExportPolicyCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NFS Export Policy created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateNFSExportPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSExportPolicy) (*mcp.CallToolResult, any, error) {
	nfsExportPolicyUpdate, err := newUpdateNFSExportPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateNFSExportPolicy(ctx, parameters.ExportPolicy, nfsExportPolicyUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NFS Export Policy updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newUpdateNFSExportPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateNFSExportPolicy(in tool.NFSExportPolicy) (ontap.ExportPolicy, error) {
	out := ontap.ExportPolicy{}
	if in.ExportPolicy == "" {
		return out, errors.New("export policy name is required")
	}
	if in.NewExportPolicy != "" {
		out.Name = in.NewExportPolicy
	}

	if in.ClientMatch != "" || in.ROrule != "" || in.RWrule != "" {
		if in.ClientMatch == "" {
			return out, errors.New("client match is required")
		}
		if in.ROrule == "" {
			return out, errors.New("read only rules are required")
		}
		if in.RWrule == "" {
			return out, errors.New("read write rules are required")
		}

		out.Rules = []ontap.Rule{
			{Clients: []ontap.ClientData{
				{Match: in.ClientMatch},
			},
				RWrule: strings.Split(in.RWrule, ","),
				ROrule: strings.Split(in.ROrule, ","),
			},
		}
	}

	return out, nil
}

func (a *App) DeleteNFSExportPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSExportPolicy) (*mcp.CallToolResult, any, error) {
	nfsExportPolicyDelete, err := newDeleteNFSExportPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteNFSExportPolicy(ctx, nfsExportPolicyDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NFS Export Policy deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newDeleteNFSExportPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteNFSExportPolicy(in tool.NFSExportPolicy) (ontap.ExportPolicy, error) {
	out := ontap.ExportPolicy{}
	if in.ExportPolicy == "" {
		return out, errors.New("export policy name is required")
	}
	out.Name = in.ExportPolicy
	return out, nil
}

// newCreateNFSExportPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateNFSExportPolicy(in tool.NFSExportPolicy) (ontap.ExportPolicy, error) {
	out := ontap.ExportPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.ExportPolicy == "" {
		return out, errors.New("nfs export policy name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.ExportPolicy

	if in.ClientMatch != "" || in.ROrule != "" || in.RWrule != "" {
		if in.ClientMatch == "" {
			return out, errors.New("client match is required")
		}
		if in.ROrule == "" {
			return out, errors.New("read only rules are required")
		}
		if in.RWrule == "" {
			return out, errors.New("read write rules are required")
		}

		out.Rules = []ontap.Rule{
			{Clients: []ontap.ClientData{
				{Match: in.ClientMatch},
			},
				RWrule: strings.Split(in.RWrule, ","),
				ROrule: strings.Split(in.ROrule, ","),
			},
		}
	}
	return out, nil
}

func (a *App) CreateNFSExportPoliciesRule(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSExportPolicyRules) (*mcp.CallToolResult, any, error) {
	nfsExportPolicyRulesCreate, err := newCreateNFSExportPolicyRules(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateNFSExportPolicyRules(ctx, parameters.ExportPolicy, nfsExportPolicyRulesCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NFS Export Policy Rules created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newCreateNFSExportPolicyRules validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateNFSExportPolicyRules(in tool.NFSExportPolicyRules) (ontap.Rule, error) {
	out := ontap.Rule{}
	if in.ClientMatch == "" {
		return out, errors.New("client match is required")
	}
	if in.ROrule == "" {
		return out, errors.New("read only rules are required")
	}
	if in.RWrule == "" {
		return out, errors.New("read write rules are required")
	}
	if in.ExportPolicy == "" {
		return out, errors.New("export policy name is required")
	}

	out.Clients = []ontap.ClientData{
		{Match: in.ClientMatch},
	}
	out.ROrule = strings.Split(in.ROrule, ",")
	out.RWrule = strings.Split(in.RWrule, ",")
	return out, nil
}

func (a *App) UpdateNFSExportPoliciesRule(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSExportPolicyRules) (*mcp.CallToolResult, any, error) {
	nfsExportPolicyRulesUpdate, err := newUpdateNFSExportPolicyRules(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateNFSExportPolicyRules(ctx, parameters.ExportPolicy, parameters.OldClientMatch, parameters.OldROrule, parameters.OldRWrule, nfsExportPolicyRulesUpdate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NFS Export Policy Rules updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newUpdateNFSExportPolicyRules validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateNFSExportPolicyRules(in tool.NFSExportPolicyRules) (ontap.Rule, error) {
	out := ontap.Rule{}
	if in.ExportPolicy == "" {
		return out, errors.New("export policy name is required")
	}
	if in.OldClientMatch == "" && in.OldRWrule == "" && in.OldROrule == "" {
		return out, errors.New("old client match OR ro rule OR rw rules are required")
	}
	if in.ClientMatch != "" {
		out.Clients = []ontap.ClientData{
			{Match: in.ClientMatch},
		}
	}
	if in.ROrule != "" {
		out.ROrule = strings.Split(in.ROrule, ",")
	}
	if in.RWrule != "" {
		out.RWrule = strings.Split(in.RWrule, ",")
	}

	return out, nil
}

func (a *App) DeleteNFSExportPoliciesRule(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.NFSExportPolicyRules) (*mcp.CallToolResult, any, error) {
	nfsExportPolicyRulesDelete, err := newDeleteNFSExportPolicyRules(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteNFSExportPolicyRules(ctx, parameters.ExportPolicy, nfsExportPolicyRulesDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "NFS Export Policy Rules deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newDeleteNFSExportPolicyRules validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteNFSExportPolicyRules(in tool.NFSExportPolicyRules) (ontap.Rule, error) {
	out := ontap.Rule{}
	if in.ClientMatch == "" && in.ROrule == "" && in.RWrule == "" {
		return out, errors.New("old client match OR ro rule OR rw rules are required")
	}

	if in.ClientMatch != "" {
		out.ClientsStr = in.ClientMatch
	}

	if in.ROrule != "" {
		out.ROruleStr = in.ROrule
	}

	if in.RWrule != "" {
		out.RWruleStr = in.RWrule
	}

	return out, nil
}
