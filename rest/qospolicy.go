package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) GetQoSPolicy(ctx context.Context, qosPolicyGet ontap.QoSPolicy) ([]string, error) {
	var (
		qosPolicy ontap.GetData
	)
	responseHeaders := http.Header{}
	qosPolicies := []string{}

	// If we only have the snapshotPolicy name we need to find the snapshotPolicy's UUID
	aClient := c.getHTTPClient()

	creds, err := c.getAuth(ctx)
	if err != nil {
		return []string{}, err
	}

	params := url.Values{}
	svmName := qosPolicyGet.SVM.Name
	if svmName != "" {
		params.Set("svm", svmName)
	}

	builder := requests.
		URL(`https://` + c.poller.Addr + `/api/storage/qos/policies`).
		Params(params).
		ToJSON(&qosPolicy).
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

	if qosPolicy.NumRecords == 0 {
		return []string{}, errors.New("no qos policies found in the cluster")
	}

	for _, qos := range qosPolicy.Records {
		qosPolicies = append(qosPolicies, qos.Name)
	}

	return qosPolicies, nil
}

func (c *Client) CreateQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, &statusCode, responseHeaders).
		BodyJSON(qosPolicy).
		ToBytesBuffer(&buf)

	err := c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		return nil
	}
	return err
}

func (c *Client) UpdateQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy, oldQosPolicyName string, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		qPolicy    ontap.GetData
	)
	responseHeaders := http.Header{}

	// If we only have the volume name we need to find the volume's UUID
	aClient := c.getHTTPClient()

	creds, err := c.getAuth(ctx)
	if err != nil {
		return err
	}

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", oldQosPolicyName)
	params.Set("svm", svmName)

	builder := requests.
		URL(`https://` + c.poller.Addr + `/api/storage/qos/policies`).
		Params(params).
		ToJSON(&qPolicy).
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

	if qPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to update qos policy %s on svm %s because it does not exist", oldQosPolicyName, svmName)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/qos/policies/`+qPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Patch().
		ToBytesBuffer(&buf).
		BodyJSON(qosPolicy)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}

func (c *Client) DeleteQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		qPolicy    ontap.GetData
	)
	responseHeaders := http.Header{}

	aClient := c.getHTTPClient()

	creds, err := c.getAuth(ctx)
	if err != nil {
		return err
	}

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", qosPolicy.Name)
	params.Set("svm", qosPolicy.SVM.Name)

	builder := requests.
		URL(`https://` + c.poller.Addr + `/api/storage/qos/policies`).
		Params(params).
		ToJSON(&qPolicy).
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

	if qPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to delete qos policy %s on svm %s because it does not exist", qosPolicy.Name, qosPolicy.SVM.Name)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/qos/policies/`+qPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}
