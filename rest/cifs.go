package rest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) CreateCIFSShare(ctx context.Context, cifsShare ontap.CIFSShare) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/cifs/shares`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		BodyJSON(cifsShare)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateCIFSShare(ctx context.Context, svmName, cifsShareName string, cifsShare ontap.CIFSShare) error {
	var (
		statusCode int
		cShare     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", cifsShareName)
	params.Set("svm", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/cifs/shares`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cShare)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if cShare.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of cifs share %s because it does not exist", cifsShareName)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/cifs/shares/`+cShare.Records[0].Svm.UUID+`/`+cifsShareName, &statusCode, responseHeaders).
		BodyJSON(cifsShare).
		ToJSON(&cShare).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteCIFSShare(ctx context.Context, cifsShare ontap.CIFSShare) error {
	var (
		statusCode int
		cShare     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", cifsShare.Name)
	params.Set("svm", cifsShare.SVM.Name)

	builder := c.baseRequestBuilder(`/api/protocols/cifs/shares`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cShare)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if cShare.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of cifs share %s because it does not exist", cifsShare.Name)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/cifs/shares/`+cShare.Records[0].Svm.UUID+`/`+cifsShare.Name, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
