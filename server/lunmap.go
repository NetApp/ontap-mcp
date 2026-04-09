package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateLunMap(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.LunMap) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	lunMapCreate, err := newCreateLunMap(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.CreateLunMap(ctx, lunMapCreate)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "lun map created successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteLunMap(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.LunMap) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := validateDeleteLunMap(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.DeleteLunMap(ctx, parameters.SVM, parameters.LunName, parameters.IGroupName)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "lun map deleted successfully"},
		},
	}, nil, nil
}

func newCreateLunMap(in tool.LunMap) (ontap.LunMap, error) {
	out := ontap.LunMap{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.LunName == "" {
		return out, errors.New("LUN name is required")
	}
	if in.IGroupName == "" {
		return out, errors.New("igroup name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Lun = ontap.NameAndUUID{Name: in.LunName}
	out.IGroup = ontap.NameAndUUID{Name: in.IGroupName}
	return out, nil
}

func validateDeleteLunMap(in tool.LunMap) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	if in.LunName == "" {
		return errors.New("LUN name is required")
	}
	if in.IGroupName == "" {
		return errors.New("igroup name is required")
	}
	return nil
}
