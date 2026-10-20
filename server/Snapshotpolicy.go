package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"strconv"
	"strings"
)

func (a *App) CreateSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicyCreate) (*mcp.CallToolResult, any, error) {
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

func (a *App) ModifySnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicyModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}
	if parameters.Name == "" {
		return nil, nil, errors.New("snapshot policy name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		snapshotPolicyUpdate, err := updateSnapshotPolicyValidation(parameters.SnapshotPolicyUpdate)
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateSnapshotPolicy(ctx, snapshotPolicyUpdate, parameters.Name, parameters.SVM)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Snapshot policy updated successfully"},
			},
		}, nil, nil
	case "delete":
		snapshotPolicyDelete := ontap.SnapshotPolicy{
			SVM:  ontap.NameAndUUID{Name: parameters.SVM},
			Name: parameters.Name,
		}

		err = client.DeleteSnapshotPolicy(ctx, snapshotPolicyDelete)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Snapshot policy deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

func updateSnapshotPolicyValidation(in tool.SnapshotPolicyUpdate) (ontap.SnapshotPolicy, error) {
	out := ontap.SnapshotPolicy{}

	hasUpdate := false
	if in.Comment != "" {
		out.Comment = in.Comment
		hasUpdate = true
	}
	if in.Enabled != "" {
		out.Enabled = in.Enabled
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: comment or enabled")
	}

	return out, nil
}

// newCreateSnapshotPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateSnapshotPolicy(in tool.SnapshotPolicyCreate) (ontap.SnapshotPolicy, error) {
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

	hasUpdate := false
	if in.Comment != "" {
		out.Comment = in.Comment
		hasUpdate = true
	}
	if in.Enabled != "" {
		out.Enabled = in.Enabled
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: comment or enabled")
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

// cronFields lists the five cron fields in order with the bounds ONTAP accepts for each.
// ONTAP numbers months 1-12 and weekdays 0-6, with 0 meaning Sunday.
var cronFields = []struct {
	name     string
	minRange int
	maxRange int
}{
	{name: "minutes", minRange: 0, maxRange: 59},
	{name: "hours", minRange: 0, maxRange: 23},
	{name: "days", minRange: 1, maxRange: 31},
	{name: "months", minRange: 1, maxRange: 12},
	{name: "weekdays", minRange: 0, maxRange: 6},
}

// readRanges parses one cron field, which is either "*" or a comma-separated list of values and
// from-to ranges, and appends every value it names to out. Any value outside minRange-maxRange,
// non-numeric or empty token, or inverted range is an error, so ONTAP never receives a schedule
// that differs from the one requested.
func readRanges(name string, minRange int, maxRange int, r string, out *[]int) error {
	if r == "*" {
		return nil
	}
	for rng := range strings.SplitSeq(r, ",") {
		fromStr, toStr, isRange := strings.Cut(rng, "-")
		from, err := readCronValue(name, minRange, maxRange, rng, fromStr)
		if err != nil {
			return err
		}
		if !isRange {
			*out = append(*out, from)
			continue
		}
		to, err := readCronValue(name, minRange, maxRange, rng, toStr)
		if err != nil {
			return err
		}
		if from > to {
			return fmt.Errorf("invalid cron %s range %q: start is greater than end", name, rng)
		}
		for i := from; i <= to; i++ {
			*out = append(*out, i)
		}
	}
	return nil
}

// readCronValue parses one number of the token rng in a cron field and checks its bounds.
func readCronValue(name string, minRange int, maxRange int, rng string, s string) (int, error) {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return 0, fmt.Errorf("invalid cron %s value %q: expected *, a number, or a range such as 1-5", name, rng)
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < minRange || v > maxRange {
		return 0, fmt.Errorf("invalid cron %s value %q: must be between %d and %d", name, rng, minRange, maxRange)
	}
	return v, nil
}

func convertCron(cronStr string, out *ontap.Schedule) error {
	fields := strings.Fields(cronStr)
	for _, field := range fields {
		if strings.Contains(field, "/") {
			return fmt.Errorf("wrong cron format %s detected", field)
		}
	}
	if len(fields) != len(cronFields) {
		return fmt.Errorf("invalid cron expression %q: expected 5 fields (minutes hours days months weekdays), got %d", cronStr, len(fields))
	}
	targets := []*[]int{&out.Cron.Minutes, &out.Cron.Hours, &out.Cron.Days, &out.Cron.Months, &out.Cron.Weekdays}
	for i, f := range cronFields {
		if err := readRanges(f.name, f.minRange, f.maxRange, fields[i], targets[i]); err != nil {
			return err
		}
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

func (a *App) ModifyScheduleInSnapshotPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SnapshotPolicyScheduleModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.SVM == "" {
		return nil, nil, errors.New("SVM name is required")
	}
	if parameters.PolicyName == "" {
		return nil, nil, errors.New("snapshot policy name is required")
	}
	if parameters.ScheduleName == "" {
		return nil, nil, errors.New("schedule name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		scheduleUpdate, err := updateScheduleInSnapshotPolicyValidation(parameters.SnapshotPolicyScheduleUpdate)
		if err != nil {
			return nil, nil, err
		}

		err = client.UpdateScheduleInSnapshotPolicy(ctx, parameters.PolicyName, parameters.SVM, parameters.ScheduleName, scheduleUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Schedule updated in snapshot policy successfully"},
			},
		}, nil, nil
	case "remove":
		err = client.RemoveScheduleInSnapshotPolicy(ctx, parameters.PolicyName, parameters.SVM, parameters.ScheduleName)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Schedule removed from snapshot policy successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, remove", parameters.Operation)), nil, nil
	}
}

func updateScheduleInSnapshotPolicyValidation(in tool.SnapshotPolicyScheduleUpdate) (ontap.SnapshotPolicySchedule, error) {
	out := ontap.SnapshotPolicySchedule{}

	hasUpdate := false
	if in.Count < 0 {
		return out, errors.New("count must be non-negative")
	}
	if in.Count > 0 {
		out.Count = in.Count
		hasUpdate = true
	}
	if in.SnapmirrorLabel != "" {
		out.SnapmirrorLabel = in.SnapmirrorLabel
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: count or snapmirror_label")
	}

	return out, nil
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

	hasUpdate := false
	if in.SnapmirrorLabel != "" {
		out.SnapmirrorLabel = in.SnapmirrorLabel
		hasUpdate = true
	}
	if in.Count > 0 {
		out.Count = in.Count
		hasUpdate = true
	}

	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: count or snapmirror_label")
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
