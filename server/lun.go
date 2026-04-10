package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func lunPath(volume, name string) string {
	return "/vol/" + volume + "/" + name
}

func (a *App) CreateLUN(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.LUN) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	lunCreate, err := newCreateLUN(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.CreateLUN(ctx, lunCreate); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "LUN created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateLUN(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.LUN) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	lunUpdate, err := newUpdateLUN(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.UpdateLUN(ctx, parameters.SVM, lunPath(parameters.Volume, parameters.Name), lunUpdate); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "LUN updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteLUN(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.LUN) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := newDeleteLUN(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := client.DeleteLUN(ctx, parameters.SVM, lunPath(parameters.Volume, parameters.Name), parameters.AllowDeleteWhileMapped); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "LUN deleted successfully"},
		},
	}, nil, nil
}

// newCreateLUN validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateLUN(in tool.LUN) (ontap.LUN, error) {
	out := ontap.LUN{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Name == "" {
		return out, errors.New("LUN name is required")
	}
	if in.Size == "" {
		return out, errors.New("LUN size is required")
	}
	if in.OsType == "" {
		return out, errors.New("OS type is required")
	}

	size, err := parseSize(in.Size)
	if err != nil {
		return out, fmt.Errorf("invalid size: %w", err)
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = lunPath(in.Volume, in.Name)
	out.Space = ontap.LUNSpace{Size: size}
	out.OsType = in.OsType
	return out, nil
}

// newUpdateLUN validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateLUN(in tool.LUN) (ontap.LUN, error) {
	out := ontap.LUN{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Name == "" {
		return out, errors.New("LUN name is required")
	}
	if in.NewName == "" && in.Size == "" && in.Enabled == "" {
		return out, errors.New("at least one of new_lun_name, size, or enabled must be provided")
	}

	if in.NewName != "" {
		out.Name = lunPath(in.Volume, in.NewName)
	}

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return out, fmt.Errorf("invalid size: %w", err)
		}
		out.Space = ontap.LUNSpace{Size: size}
	}

	if in.Enabled != "" {
		enabled, err := strconv.ParseBool(in.Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid enabled value %q: must be 'true' or 'false'", in.Enabled)
		}
		out.Enabled = &enabled
	}

	return out, nil
}

// newDeleteLUN validates the customer provided arguments for a LUN delete operation
func newDeleteLUN(in tool.LUN) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return errors.New("volume name is required")
	}
	if in.Name == "" {
		return errors.New("LUN name is required")
	}
	return nil
}
