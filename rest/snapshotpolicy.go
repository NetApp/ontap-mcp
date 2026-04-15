package rest

import (
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) CreateSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) error {
	var (
		statusCode   int
		scheduleName string
		oc           ontap.OnlyCount
		err          error
	)
	responseHeaders := http.Header{}

	// If schedule is exist then use it else create new
	if len(snapshotPolicy.Copies) > 0 {
		scheduleName = snapshotPolicy.Copies[0].Schedule.Name
	}
	if scheduleName == "" {
		return fmt.Errorf("schedule name must be required in snapshot policy %s", snapshotPolicy.Name)
	}
	params := url.Values{}
	params.Set("return_records", "false")
	params.Set("fields", "name")
	params.Set("name", scheduleName)

	builder := c.baseRequestBuilder(`/api/cluster/schedules`, &statusCode, responseHeaders).
		ToJSON(&oc).
		Params(params)

	err = c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if oc.NumRecords == 0 {
		return fmt.Errorf("no schedule %s found", scheduleName)
	} else if oc.NumRecords != 1 {
		return fmt.Errorf("failed to create snapshotPolicy=%s on svm=%s with given schedule name=%s because there are %d matching schedules",
			snapshotPolicy.Name, snapshotPolicy.SVM.Name, scheduleName, oc.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/snapshot-policies`, &statusCode, responseHeaders).
		BodyJSON(snapshotPolicy)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy, snapshotPolicyName string, svmName string) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}
	policyUUID, err := c.getSnapshotPolicyUUID(ctx, snapshotPolicyName, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+policyUUID, &statusCode, responseHeaders).
		BodyJSON(snapshotPolicy).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}
	policyUUID, err := c.getSnapshotPolicyUUID(ctx, snapshotPolicy.Name, snapshotPolicy.SVM.Name)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+policyUUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) CreateSchedule(ctx context.Context, schedule ontap.Schedule) error {
	var statusCode int

	builder := c.baseRequestBuilder(`/api/cluster/schedules`, &statusCode, nil).
		BodyJSON(schedule)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) getSnapshotPolicyUUID(ctx context.Context, policyName, svmName string) (string, error) {
	var ssPolicy ontap.GetData
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", policyName)
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&ssPolicy)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if ssPolicy.NumRecords == 0 {
		return "", fmt.Errorf("failed to get snapshot policy=%s on svm=%s because it does not exist", policyName, svmName)
	}
	if ssPolicy.NumRecords != 1 {
		return "", fmt.Errorf("failed to get snapshot policy=%s on svm=%s because there are %d matching records", policyName, svmName, ssPolicy.NumRecords)
	}

	return ssPolicy.Records[0].UUID, nil
}

func (c *Client) getSnapshotPolicyScheduleUUID(ctx context.Context, policyUUID, scheduleName string) (string, error) {
	var scheduleRecords ontap.GetData
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "schedule.uuid,schedule.name")
	params.Set("schedule.name", scheduleName)

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+policyUUID+`/schedules`, nil, responseHeaders).
		Params(params).
		ToJSON(&scheduleRecords)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if scheduleRecords.NumRecords == 0 {
		return "", fmt.Errorf("schedule=%s does not exist in snapshot policy uuid=%s", scheduleName, policyUUID)
	}
	if scheduleRecords.NumRecords != 1 {
		return "", fmt.Errorf("found %d schedules matching name=%s in snapshot policy uuid=%s", scheduleRecords.NumRecords, scheduleName, policyUUID)
	}

	return scheduleRecords.Records[0].Schedule.UUID, nil
}

func (c *Client) AddScheduleInSnapshotPolicy(ctx context.Context, policyName, svmName string, schedule ontap.SnapshotPolicySchedule) error {
	var statusCode int
	responseHeaders := http.Header{}

	policyUUID, err := c.getSnapshotPolicyUUID(ctx, policyName, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+policyUUID+`/schedules`, &statusCode, responseHeaders).
		BodyJSON(schedule)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateScheduleInSnapshotPolicy(ctx context.Context, policyName, svmName, scheduleName string, schedule ontap.SnapshotPolicySchedule) error {
	var statusCode int
	responseHeaders := http.Header{}

	policyUUID, err := c.getSnapshotPolicyUUID(ctx, policyName, svmName)
	if err != nil {
		return err
	}

	scheduleUUID, err := c.getSnapshotPolicyScheduleUUID(ctx, policyUUID, scheduleName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+policyUUID+`/schedules/`+scheduleUUID, &statusCode, responseHeaders).
		BodyJSON(schedule).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) RemoveScheduleInSnapshotPolicy(ctx context.Context, policyName, svmName, scheduleName string) error {
	var statusCode int
	responseHeaders := http.Header{}

	policyUUID, err := c.getSnapshotPolicyUUID(ctx, policyName, svmName)
	if err != nil {
		return err
	}

	scheduleUUID, err := c.getSnapshotPolicyScheduleUUID(ctx, policyUUID, scheduleName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+policyUUID+`/schedules/`+scheduleUUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
