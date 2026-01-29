package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
	"time"
)

func (c *Client) DeleteSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		ssPolicy   ontap.GetData
	)
	responseHeaders := http.Header{}

	creds, err := c.getAuth(ctx)
	if err != nil {
		return err
	}

	aClient := c.getHTTPClient()

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", snapshotPolicy.Name)

	builder := requests.
		URL(`https://` + c.poller.Addr + `/api/storage/snapshot-policies`).
		Params(params).
		ToJSON(&ssPolicy).
		Client(aClient).
		CopyHeaders(responseHeaders).
		AddValidator(func(response *http.Response) error {
			statusCode = response.StatusCode
			return nil
		}).
		AddValidator(ontapValidator)

	if creds.AuthToken != "" {
		builder = builder.Bearer(creds.AuthToken)
	} else {
		builder = builder.BasicAuth(creds.Username, creds.Password)
	}

	err = builder.Fetch(ctx)

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

	builder = c.baseRequestBuilder(`/api/storage/snapshot-policies/`+ssPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		var pj ontap.PostJob
		err := json.Unmarshal(buf.Bytes(), &pj)
		if err != nil {
			return err
		}

		err = c.waitForJob(ctx, `/api/cluster/jobs/`+pj.Job.UUID, 3*time.Minute)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) ([]string, error) {
	var (
		ssPolicy ontap.GetData
	)
	responseHeaders := http.Header{}
	snapshotPolicies := []string{}

	creds, err := c.getAuth(ctx)
	if err != nil {
		return []string{}, err
	}

	// If we only have the snapshotPolicy name we need to find the snapshotPolicy's UUID
	aClient := c.getHTTPClient()

	params := url.Values{}
	svmName := snapshotPolicy.SVM.Name
	if svmName != "" {
		params.Set("svm", svmName)
	}

	builder := requests.
		URL(`https://` + c.poller.Addr + `/api/storage/snapshot-policies`).
		Params(params).
		ToJSON(&ssPolicy).
		Client(aClient).
		CopyHeaders(responseHeaders).
		AddValidator(func(_ *http.Response) error {
			return nil
		}).
		AddValidator(ontapValidator)

	if creds.AuthToken != "" {
		builder = builder.Bearer(creds.AuthToken)
	} else {
		builder = builder.BasicAuth(creds.Username, creds.Password)
	}

	err = builder.Fetch(ctx)

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
		return fmt.Errorf("no snapshotPolicy %s found", scheduleName)
	} else if oc.NumRecords != 1 {
		return fmt.Errorf("failed to create snapshotPolicy=%s on svm=%s with given schedule name=%s because there are %d matching export policies",
			snapshotPolicy.Name, snapshotPolicy.SVM.Name, scheduleName, oc.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/storage/snapshot-policies`, &statusCode, responseHeaders).
		BodyJSON(snapshotPolicy).
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		return nil
	}

	return err
}
