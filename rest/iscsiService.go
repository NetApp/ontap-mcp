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

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) UpdateIscsiService(ctx context.Context, svmName string, iscsiService ontap.IscsiService) error {
	var (
		statusCode int
		cShare     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/iscsi/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cShare)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if cShare.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of cifs share in svm %s because it does not exist", svmName)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/iscsi/services/`+cShare.Records[0].Svm.UUID, &statusCode, responseHeaders).
		BodyJSON(iscsiService).
		ToJSON(&cShare).
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
		cShare     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm", iscsiService.SVM.Name)

	builder := c.baseRequestBuilder(`/api/protocols/san/iscsi/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cShare)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if cShare.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of cifs share in svm %s because it does not exist", iscsiService.SVM.Name)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/iscsi/services/`+cShare.Records[0].Svm.UUID, &statusCode, responseHeaders).
		Delete()

	err = c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}
