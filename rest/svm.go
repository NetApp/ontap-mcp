package rest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) CreateSVM(ctx context.Context, svm ontap.SVMCreate) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		BodyJSON(svm).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) UpdateSVM(ctx context.Context, svm ontap.SVM, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		svmData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if svmData.NumRecords == 0 {
		return fmt.Errorf("failed to get details of SVM %s because it does not exist", svmName)
	}
	if svmData.NumRecords != 1 {
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records",
			svmName, svmData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/svm/svms/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		Patch().
		ToBytesBuffer(&buf).
		BodyJSON(svm)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteSVM(ctx context.Context, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		svmData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if svmData.NumRecords == 0 {
		return fmt.Errorf("failed to get details of SVM %s because it does not exist", svmName)
	}
	if svmData.NumRecords != 1 {
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records",
			svmName, svmData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/svm/svms/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteSVMPeer(ctx context.Context, svmName string) error {
	var (
		buf         bytes.Buffer
		statusCode  int
		svmPeerData ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/peers`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmPeerData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if svmPeerData.NumRecords == 0 {
		return fmt.Errorf("failed to get details of SVM peer %s because it does not exist", svmName)
	}
	if svmPeerData.NumRecords != 1 {
		return fmt.Errorf("failed to get details of SVM peer %s because there are %d matching records",
			svmName, svmPeerData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/svm/peers/`+svmPeerData.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}
