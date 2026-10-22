package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9_.-]*$`)

func lunPath(volume, name string, remote ontap.Remote) string {
	if remote.Model == ontap.ASAr2 {
		return name
	}
	return "/vol/" + volume + "/" + name
}

func (a *App) CreateLUN(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.LUNCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	remote, err := a.getClusterRemote(ctx, parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	lunCreate, err := newCreateLUN(parameters, remote)
	if err != nil {
		return nil, nil, err
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

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	remote, err := a.getClusterRemote(ctx, parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	lunUpdate, err := newUpdateLUN(parameters, remote)
	if err != nil {
		return nil, nil, err
	}

	if err := client.UpdateLUN(ctx, parameters.SVM, lunPath(parameters.Volume, parameters.Name, remote), lunUpdate); err != nil {
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

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	remote, err := a.getClusterRemote(ctx, parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := validateLUN(parameters.SVM, parameters.Volume, parameters.Name, remote); err != nil {
		return nil, nil, err
	}

	if err := client.DeleteLUN(ctx, parameters.SVM, lunPath(parameters.Volume, parameters.Name, remote), parameters.AllowDeleteWhileMapped); err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "LUN deleted successfully"},
		},
	}, nil, nil
}

func (a *App) ModifyLUN(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.LUNModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	remote, err := a.getClusterRemote(ctx, parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	if err := validateLUN(parameters.SVM, parameters.Volume, parameters.Name, remote); err != nil {
		return nil, nil, err
	}

	switch parameters.Operation {
	case "update":
		lunUpdate, err := newUpdateLUN(tool.LUN{
			SVM:     parameters.SVM,
			Volume:  parameters.Volume,
			Name:    parameters.Name,
			NewName: parameters.LUNUpdate.NewName,
			Size:    parameters.LUNUpdate.Size,
			Enabled: parameters.LUNUpdate.Enabled,
		}, remote)
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateLUN(ctx, parameters.SVM, lunPath(parameters.Volume, parameters.Name, remote), lunUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "LUN updated successfully"}}}, nil, nil
	case "delete":
		err = client.DeleteLUN(ctx, parameters.SVM, lunPath(parameters.Volume, parameters.Name, remote), parameters.AllowDeleteWhileMapped)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "LUN deleted successfully"}}}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

// newCreateLUN validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateLUN(in tool.LUNCreate, remote ontap.Remote) (ontap.LUN, error) {
	out := ontap.LUN{}

	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" && remote.Model != ontap.ASAr2 {
		return out, errors.New("volume name is required")
	}
	if err := validateName(in.Volume); err != nil {
		return out, err
	}
	if in.Name == "" {
		return out, errors.New("LUN name is required")
	}
	if err := validateName(in.Name); err != nil {
		return out, err
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
	out.Name = lunPath(in.Volume, in.Name, remote)
	out.Space = ontap.LUNSpace{Size: size}
	if in.SpaceGuaranteeRequested {
		t := true
		out.Space.Guarantee = ontap.LUNSpaceGuarantee{Requested: &t}
	}
	out.OsType = in.OsType
	return out, nil
}

// newUpdateLUN validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateLUN(in tool.LUN, remote ontap.Remote) (ontap.LUN, error) {
	out := ontap.LUN{}
	hasUpdate := false
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" && remote.Model != ontap.ASAr2 {
		return out, errors.New("volume name is required")
	}
	if err := validateName(in.Volume); err != nil {
		return out, err
	}
	if in.Name == "" {
		return out, errors.New("LUN name is required")
	}
	if err := validateName(in.Name); err != nil {
		return out, err
	}
	if in.NewName == "" && in.Size == "" && in.Enabled == "" {
		return out, errors.New("at least one of new_lun_name, size, or enabled must be provided")
	}

	if in.NewName != "" {
		if err := validateName(in.NewName); err != nil {
			return out, err
		}
		out.Name = lunPath(in.Volume, in.NewName, remote)
		hasUpdate = true
	}

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return out, fmt.Errorf("invalid size: %w", err)
		}
		out.Space = ontap.LUNSpace{Size: size}
		hasUpdate = true
	}

	if in.Enabled != "" {
		enabled, err := strconv.ParseBool(in.Enabled)
		if err != nil {
			return out, fmt.Errorf("invalid enabled value %q: must be 'true' or 'false'", in.Enabled)
		}
		out.Enabled = &enabled
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name, size, or enabled")
	}

	return out, nil
}

func validateLUN(svm, volume, name string, remote ontap.Remote) error {
	if svm == "" {
		return errors.New("SVM name is required")
	}
	if volume == "" && remote.Model != ontap.ASAr2 {
		return errors.New("volume name is required")
	}
	if err := validateName(volume); err != nil {
		return err
	}
	if name == "" {
		return errors.New("LUN name is required")
	}
	if err := validateName(name); err != nil {
		return err
	}
	return nil
}

func validateName(name string) error {
	if !validName.MatchString(name) || strings.Contains(name, "..") {
		return fmt.Errorf("%s contains an invalid character or sequence; names may only contain A-Z, a-z, 0-9, _, ., and -, and cannot contain `..`", name)
	}
	return nil
}
