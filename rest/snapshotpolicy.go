package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) DeleteSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		ssPolicy   ontap.GetData
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", snapshotPolicy.Name)

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&ssPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if ssPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to delete snapshotPolicy=%s on svm=%s because it does not exist", snapshotPolicy.Name, snapshotPolicy.SVM.Name)
	}
	if ssPolicy.NumRecords != 1 {
		return fmt.Errorf("failed to delete snapshotPolicy=%s on svm=%s because there are %d matching records",
			snapshotPolicy.Name, snapshotPolicy.SVM.Name, ssPolicy.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+ssPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) GetSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) ([]string, error) {
	var (
		ssPolicy ontap.GetData
	)
	responseHeaders := http.Header{}
	snapshotPolicies := []string{}
	params := url.Values{}
	svmName := snapshotPolicy.SVM.Name
	if svmName != "" {
		params.Set("svm", svmName)
	}

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&ssPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return []string{}, err
	}

	if ssPolicy.NumRecords == 0 {
		if svmName != "" {
			return []string{}, fmt.Errorf("no snapshot policies found on svm: %s", svmName)
		}
		return []string{}, errors.New("no snapshot policies found in the cluster")
	}

	for _, ss := range ssPolicy.Records {
		snapshotPolicies = append(snapshotPolicies, ss.Name)
	}

	return snapshotPolicies, nil
}

func (c *Client) CreateSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		oc         ontap.OnlyCount
		err        error
	)
	responseHeaders := http.Header{}

	// If schedule is exist then use it else create new
	scheduleName := snapshotPolicy.Copies[0].Schedule.Name
	if scheduleName == "" {
		return fmt.Errorf("no schedule exist with %s name", scheduleName)
	}
	params := url.Values{}
	params.Set("return_records", "false")
	params.Set("fields", "name")
	params.Set("name", scheduleName)

	builder := c.baseRequestBuilder(`/api/cluster/schedules`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
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
		BodyJSON(snapshotPolicy).
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		return nil
	}

	return err
}

func (c *Client) CreateSchedule(ctx context.Context, schedule ontap.Schedule) error {
	var statusCode int

	builder := c.baseRequestBuilder(`/api/cluster/schedules`, &statusCode, nil).
		BodyJSON(schedule)

	err := c.buildAndExecuteRequest(ctx, builder)
	if err != nil {
		return err
	}

	if statusCode != http.StatusCreated && statusCode != http.StatusAccepted {
		return fmt.Errorf(`failed to create schedule %s: unexpected status code: %d`, schedule.Name, statusCode)
	}

	return nil
}
