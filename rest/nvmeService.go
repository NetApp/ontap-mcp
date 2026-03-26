package rest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) CreateNVMeService(ctx context.Context, nvmeService ontap.NVMeService) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nvme/services`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		BodyJSON(nvmeService)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNVMeService(ctx context.Context, svmName string, nvmeService ontap.NVMeService) error {
	var (
		statusCode int
		nvmeSr     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme service in svm %s because it does not exist", svmName)
	}

	builder = c.baseRequestBuilder(`/api/protocols/nvme/services/`+nvmeSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		BodyJSON(nvmeService).
		ToJSON(&nvmeSr).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNVMeService(ctx context.Context, nvmeService ontap.NVMeService) error {
	var (
		statusCode int
		nvmeSr     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm", nvmeService.SVM.Name)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme service in svm %s because it does not exist", nvmeService.SVM.Name)
	}

	builder = c.baseRequestBuilder(`/api/protocols/nvme/services/`+nvmeSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
