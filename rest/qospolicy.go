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

func (c *Client) GetQoSPolicy(ctx context.Context, qosPolicyGet ontap.QoSPolicy) ([]string, error) {
	var (
		qosPolicy ontap.GetData
	)
	responseHeaders := http.Header{}
	qosPolicies := []string{}
	params := url.Values{}
	svmName := qosPolicyGet.SVM.Name
	if svmName != "" {
		params.Set("svm", svmName)
	}

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&qosPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

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

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}
