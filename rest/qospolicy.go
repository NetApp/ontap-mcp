package rest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) CreateQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, &statusCode, responseHeaders).
		BodyJSON(qosPolicy).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy, oldQosPolicyName string, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		qPolicy    ontap.GetData
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", oldQosPolicyName)
	params.Set("svm", svmName)

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&qPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

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

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		qPolicy    ontap.GetData
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", qosPolicy.Name)
	params.Set("svm", qosPolicy.SVM.Name)

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&qPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if qPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to delete qos policy %s on svm %s because it does not exist", qosPolicy.Name, qosPolicy.SVM.Name)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/qos/policies/`+qPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
