package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"strings"
)

func (a *App) CreateSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicy) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

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

func (a *App) UpdateSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicy) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	snapshotPolicyUpdate, err := newUpdateSnapshotPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateSnapshotPolicy(ctx, snapshotPolicyUpdate, parameters.Name, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Snapshot policy updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicy) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

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

// newUpdateSnapshotPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateSnapshotPolicy(in tool.SnapshotPolicy) (ontap.SnapshotPolicy, error) {
	out := ontap.SnapshotPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("snapshot policy name is required")
	}

	if in.Comment != "" {
		out.Comment = in.Comment
	}
	if in.Enabled != "" {
		out.Enabled = in.Enabled
	}

	if out.Enabled == "" && out.Comment == "" {
		return out, errors.New("at least one supported update field must be provided; enabled and comment are supported for update")
	}

	return out, nil
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

func (a *App) CreateSchedule(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Schedule) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	scheduleCreate, err := newCreateSchedule(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateSchedule(ctx, scheduleCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Schedule created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newCreateSchedule validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateSchedule(in tool.Schedule) (ontap.Schedule, error) {
	out := ontap.Schedule{
		Cron: ontap.Cron{
			Minutes:  make([]int, 0),
			Hours:    make([]int, 0),
			Days:     make([]int, 0),
			Months:   make([]int, 0),
			Weekdays: make([]int, 0),
		},
	}

	if in.Name == "" {
		return out, errors.New("schedule name is required")
	}
	if in.CronExpression == "" {
		return out, errors.New("schedule cron expression is required")
	}

	out.Name = in.Name

	if err := convertCron(in.CronExpression, &out); err != nil {
		return out, err
	}

	return out, nil
}

func readRanges(minRange int, maxRange int, r string, out *[]int) {
	if r != "*" {
		for rng := range strings.SplitSeq(r, ",") {
			from, to := 0, 0
			n, _ := fmt.Sscanf(rng, "%d-%d", &from, &to)
			switch n {
			case 1: // single value
				if from >= minRange && from <= maxRange {
					*out = append(*out, from)
				}
			case 2: // range
				if from < minRange {
					from = minRange
				}
				if to > maxRange {
					to = maxRange
				}
				for i := from; i <= to; i++ {
					*out = append(*out, i)
				}
			default:
				continue
			}
		}
	}
}

func convertCron(cronStr string, out *ontap.Schedule) error {
	fields := strings.Fields(cronStr)
	for i := range 5 {
		var field string
		if i < len(fields) {
			field = fields[i]
			if strings.Contains(field, "/") {
				return fmt.Errorf("wrong cron format %s detected", field)
			}
		} else {
			// Cron misses a field, using '*'
			field = "*"
		}
		switch i {
		case 0:
			readRanges(0, 59, field, &out.Cron.Minutes)
		case 1:
			readRanges(0, 23, field, &out.Cron.Hours)
		case 2:
			readRanges(1, 31, field, &out.Cron.Days)
		case 3:
			readRanges(1, 12, field, &out.Cron.Months)
		case 4:
			readRanges(0, 6, field, &out.Cron.Weekdays)
		default:
		}
	}
	if len(fields) > 5 {
		fmt.Println("Ignoring extra fields in cron")
	}
	return nil
}

func (a *App) AddScheduleInSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicySchedule) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	scheduleEntry, err := newAddScheduleInSnapshotPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.AddScheduleInSnapshotPolicy(ctx, parameters.PolicyName, parameters.SVM, scheduleEntry)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Schedule added to snapshot policy successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateScheduleInSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicySchedule) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	scheduleEntry, err := newUpdateScheduleInSnapshotPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.UpdateScheduleInSnapshotPolicy(ctx, parameters.PolicyName, parameters.SVM, parameters.ScheduleName, scheduleEntry)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Schedule in snapshot policy updated successfully"},
		},
	}, nil, nil
}

func (a *App) RemoveScheduleInSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicySchedule) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if err := validateDeleteScheduleInSnapshotPolicy(parameters); err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.RemoveScheduleInSnapshotPolicy(ctx, parameters.PolicyName, parameters.SVM, parameters.ScheduleName)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Schedule removed from snapshot policy successfully"},
		},
	}, nil, nil
}

// newAddScheduleInSnapshotPolicy validates and converts input for adding a schedule to a snapshot policy
func newAddScheduleInSnapshotPolicy(in tool.SnapshotPolicySchedule) (ontap.SnapshotPolicySchedule, error) {
	out := ontap.SnapshotPolicySchedule{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.PolicyName == "" {
		return out, errors.New("snapshot policy name is required")
	}
	if in.ScheduleName == "" {
		return out, errors.New("schedule name is required")
	}
	if in.Count == 0 {
		return out, errors.New("snapshot copies count is required")
	}

	out.Schedule = ontap.NameAndUUID{Name: in.ScheduleName}
	out.Count = in.Count

	if in.SnapmirrorLabel != "" {
		out.SnapmirrorLabel = in.SnapmirrorLabel
	}

	return out, nil
}

// newUpdateScheduleInSnapshotPolicy validates and converts input for updating a schedule in a snapshot policy
func newUpdateScheduleInSnapshotPolicy(in tool.SnapshotPolicySchedule) (ontap.SnapshotPolicySchedule, error) {
	out := ontap.SnapshotPolicySchedule{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.PolicyName == "" {
		return out, errors.New("snapshot policy name is required")
	}
	if in.ScheduleName == "" {
		return out, errors.New("schedule name is required")
	}
	if in.Count == 0 && in.SnapmirrorLabel == "" {
		return out, errors.New("at least one supported update field must be provided; count and snapmirror_label are supported for update")
	}

	if in.SnapmirrorLabel != "" {
		out.SnapmirrorLabel = in.SnapmirrorLabel
	}
	if in.Count > 0 {
		out.Count = in.Count
	}

	return out, nil
}

// validateDeleteScheduleInSnapshotPolicy validates input for removing a schedule from a snapshot policy
func validateDeleteScheduleInSnapshotPolicy(in tool.SnapshotPolicySchedule) error {
	if in.SVM == "" {
		return errors.New("SVM name is required")
	}
	if in.PolicyName == "" {
		return errors.New("snapshot policy name is required")
	}
	if in.ScheduleName == "" {
		return errors.New("schedule name is required")
	}
	return nil
}
