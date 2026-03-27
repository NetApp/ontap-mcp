package rest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) CreateIscsiService(ctx context.Context, iscsiService ontap.IscsiService) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/san/iscsi/services`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		BodyJSON(iscsiService)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateIscsiService(ctx context.Context, svmName string, iscsiService ontap.IscsiService) error {
	var (
		statusCode int
		iscsiSr    ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/iscsi/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&iscsiSr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if iscsiSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of iscsi service in svm %s because it does not exist", svmName)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/iscsi/services/`+iscsiSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		BodyJSON(iscsiService).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteIscsiService(ctx context.Context, svmName string) error {
	var (
		statusCode int
		iscsiSr    ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/iscsi/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&iscsiSr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if iscsiSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of iscsi service in svm %s because it does not exist", svmName)
	}

	// Disable iscsi service first before delete
	iscsiService := ontap.IscsiService{Enabled: "false"}
	builder = c.baseRequestBuilder(`/api/protocols/san/iscsi/services/`+iscsiSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		BodyJSON(iscsiService).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if err := c.checkStatus(statusCode); err != nil {
		return err
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/iscsi/services/`+iscsiSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
