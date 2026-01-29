package server

import (
	"context"
	"errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"strings"
)

func (a *App) ListSnapshotPolicies(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicy) (*mcp.CallToolResult, any, error) {
	snapshotPolicyGet := newGetSnapshotPolicy(parameters)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	snapshotPolicies, err := client.GetSnapshotPolicy(ctx, snapshotPolicyGet)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(snapshotPolicies, ",")},
		},
	}, nil, nil
}

func (a *App) DeleteSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicy) (*mcp.CallToolResult, any, error) {
	snapshotPolicyDelete, err := newDeleteSnapshotPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteSnapshotPolicy(ctx, snapshotPolicyDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Snapshot policy deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) CreateSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicy) (*mcp.CallToolResult, any, error) {
	snapshotPolicyCreate, err := newCreateSnapshotPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateSnapshotPolicy(ctx, snapshotPolicyCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Snapshot policy created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newDeleteSnapshotPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteSnapshotPolicy(in tool.SnapshotPolicy) (ontap.SnapshotPolicy, error) {
	out := ontap.SnapshotPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("snapshot policy name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name

	return out, nil
}

// newGetSnapshotPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newGetSnapshotPolicy(in tool.SnapshotPolicy) ontap.SnapshotPolicy {
	out := ontap.SnapshotPolicy{}
	if in.SVM != "" {
		out.SVM = ontap.NameAndUUID{Name: in.SVM}
	}

	return out
}

// newCreateSnapshotPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateSnapshotPolicy(in tool.SnapshotPolicy) (ontap.SnapshotPolicy, error) {
	out := ontap.SnapshotPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("snapshot policy name is required")
	}
	if in.Schedule == "" {
		return out, errors.New("schedule is required")
	}
	if in.Count == 0 {
		return out, errors.New("snapshot copies count is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	out.Copies = []ontap.Copy{
		{
			Count:    in.Count,
			Schedule: ontap.Schedule{Name: in.Schedule},
		},
	}

	return out, nil
}
