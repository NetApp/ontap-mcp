package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateFCPService(ctx context.Context, fcpService ontap.FCPService) error {
	var statusCode int

	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/san/fcp/services`, &statusCode, responseHeaders).
		BodyJSON(fcpService)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateFCPService(ctx context.Context, svmName string, fcpService ontap.FCPService) error {
	var (
		statusCode int
		fcpSr      ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/fcp/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&fcpSr)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if fcpSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of fcp service in svm %s because it does not exist", svmName)
	}

	if fcpSr.NumRecords != 1 {
		return fmt.Errorf("failed to get fcp service on svm=%s because there are %d matching records",
			svmName, fcpSr.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/fcp/services/`+fcpSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		BodyJSON(fcpService).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteFCPService(ctx context.Context, svmName string) error {
	var (
		statusCode int
		fcpSr      ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/fcp/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&fcpSr)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if fcpSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of fcp service in svm %s because it does not exist", svmName)
	}

	if fcpSr.NumRecords != 1 {
		return fmt.Errorf("failed to get fcp service on svm=%s because there are %d matching records",
			svmName, fcpSr.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/fcp/services/`+fcpSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) CreateFCInterface(ctx context.Context, fcInterface ontap.FCInterface) error {
	var statusCode int

	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/network/fc/interfaces`, &statusCode, responseHeaders).
		BodyJSON(fcInterface)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateFCInterface(ctx context.Context, svmName string, name string, fcInterface ontap.FCInterface) error {
	var (
		statusCode int
		fcIfSr     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)

	builder := c.baseRequestBuilder(`/api/network/fc/interfaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&fcIfSr)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if fcIfSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of fc interface %s in svm %s because it does not exist", name, svmName)
	}

	if fcIfSr.NumRecords != 1 {
		return fmt.Errorf("failed to get unique fc interface %s in svm=%s because there are %d matching records",
			name, svmName, fcIfSr.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/network/fc/interfaces/`+fcIfSr.Records[0].UUID, &statusCode, responseHeaders).
		BodyJSON(fcInterface).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteFCInterface(ctx context.Context, svmName string, name string) error {
	var (
		statusCode int
		fcIfSr     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)

	builder := c.baseRequestBuilder(`/api/network/fc/interfaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&fcIfSr)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if fcIfSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of fc interface %s in svm %s because it does not exist", name, svmName)
	}

	if fcIfSr.NumRecords != 1 {
		return fmt.Errorf("failed to delete fc interface %s in svm=%s because there are %d matching records",
			name, svmName, fcIfSr.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/network/fc/interfaces/`+fcIfSr.Records[0].UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
