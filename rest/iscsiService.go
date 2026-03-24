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

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if statusCode == http.StatusCreated {
		return nil
	}
	return err
}

func (c *Client) UpdateIscsiService(ctx context.Context, svmName string, iscsiService ontap.IscsiService) error {
	var (
		statusCode int
		iscsiSr    ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm", svmName)

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
		ToJSON(&iscsiSr).
		Patch()

	err = c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}

func (c *Client) DeleteIscsiService(ctx context.Context, iscsiService ontap.IscsiService) error {
	var (
		statusCode int
		iscsiSr    ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm", iscsiService.SVM.Name)

	builder := c.baseRequestBuilder(`/api/protocols/san/iscsi/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&iscsiSr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if iscsiSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of iscsi service in svm %s because it does not exist", iscsiService.SVM.Name)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/iscsi/services/`+iscsiSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		Delete()

	err = c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}
