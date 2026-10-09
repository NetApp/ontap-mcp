package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateStorageUnit(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.StorageUnitCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	remote := ontap.Remote{Model: ontap.CDOT}
	if info, err := a.getClusterRemote(ctx, parameters.Cluster); err == nil {
		remote = info
		if remote.Model == "" {
			remote.Model = ontap.CDOT
		}
	} else {
		return errorResult(err), nil, err
	}

	storageUnit, err := newCreateStorageUnit(parameters, remote)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateStorageUnit(ctx, storageUnit)
	if err != nil {
		return errorResult(err), nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Storage unit created successfully"}},
	}, nil, nil
}

func (a *App) ModifyStorageUnit(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.StorageUnitModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if strings.TrimSpace(parameters.Name) == "" {
		return nil, nil, errors.New("storage unit name is required")
	}
	if err := validateStorageUnitName(parameters.Name, "storage unit name"); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(parameters.SVM) == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	switch parameters.Operation {
	case "update":
		if err := validateStorageUnitName(parameters.StorageUnitUpdate.NewName, "new storage unit name"); err != nil {
			return errorResult(err), nil, err
		}
		storageUnit, err := newUpdateStorageUnit(parameters.StorageUnitUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}
		if err := client.UpdateStorageUnit(ctx, storageUnit, parameters.Name, parameters.SVM); err != nil {
			return errorResult(err), nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Storage unit updated successfully"}},
		}, nil, nil
	case "delete":
		if err := client.DeleteStorageUnit(ctx, parameters.Name, parameters.SVM); err != nil {
			return errorResult(err), nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Storage unit deleted successfully"}},
		}, nil, nil
	default:
		err := fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)
		return errorResult(err), nil, nil
	}
}

func validateStorageUnitName(name, field string) error {
	if strings.ContainsAny(name, `*|!<>{}`) {
		return fmt.Errorf("%s contains an invalid character; names cannot contain *|!<>{}", field)
	}
	return nil
}

func newCreateStorageUnit(in tool.StorageUnitCreate, remote ontap.Remote) (ontap.StorageUnit, error) {
	if remote.Model != ontap.ASAr2 {
		return ontap.StorageUnit{}, errors.New("storage unit creation is only supported on ASAr2 clusters")
	}

	if strings.TrimSpace(in.Name) == "" {
		return ontap.StorageUnit{}, errors.New("storage unit name is required")
	}
	if strings.TrimSpace(in.SVM) == "" {
		return ontap.StorageUnit{}, errors.New("SVM name is required")
	}
	if strings.TrimSpace(in.OsType) == "" {
		return ontap.StorageUnit{}, errors.New("OS type is required")
	}
	size, err := parseSize(in.Size)
	if err != nil {
		return ontap.StorageUnit{}, fmt.Errorf("invalid storage unit size: %w", err)
	}
	if size <= 0 {
		return ontap.StorageUnit{}, errors.New("storage unit size must be greater than zero")
	}

	return ontap.StorageUnit{
		SVM:    ontap.NameAndUUID{Name: in.SVM},
		Name:   in.Name,
		OsType: in.OsType,
		Space:  ontap.StorageUnitSpace{Size: size},
	}, nil
}

func newUpdateStorageUnit(in tool.StorageUnitUpdate) (ontap.StorageUnit, error) {
	out := ontap.StorageUnit{}
	if in.NewName != "" {
		out.Name = in.NewName
	}
	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.StorageUnit{}, fmt.Errorf("invalid storage unit size: %w", err)
		}
		if size <= 0 {
			return ontap.StorageUnit{}, errors.New("storage unit size must be greater than zero")
		}
		out.Space.Size = size
	}
	if in.OsType != "" {
		out.OsType = in.OsType
	}
	if out.Name == "" && out.Space.Size == 0 && out.OsType == "" {
		return ontap.StorageUnit{}, errors.New("at least one storage unit update field is required")
	}
	return out, nil
}
