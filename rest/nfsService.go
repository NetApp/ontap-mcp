package rest

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateNFSService(ctx context.Context, nfsService ontap.NFSService) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/services`, &statusCode, responseHeaders).
		BodyJSON(nfsService).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNFSService(ctx context.Context, svmName string, nfsService ontap.NFSService) error {
	var (
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
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records", svmName, svmData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nfs/services/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		BodyJSON(nfsService).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNFSService(ctx context.Context, svmName string) error {
	var (
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
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records", svmName, svmData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nfs/services/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
