package rest

import (
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) CreateIscsiService(ctx context.Context, iscsiService ontap.IscsiService) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/san/iscsi/services`, &statusCode, responseHeaders).
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

	builder = c.baseRequestBuilder(`/api/protocols/san/iscsi/services/`+iscsiSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) CreateNetworkIPInterface(ctx context.Context, nwInterface ontap.NetworkIPInterface) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/network/ip/interfaces`, &statusCode, responseHeaders).
		BodyJSON(nwInterface)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNetworkIPInterface(ctx context.Context, scope string, interfaceName string, svmName string, nwInterface ontap.NetworkIPInterface) error {
	var (
		statusCode    int
		interfaceData ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", interfaceName)
	if scope != "" {
		params.Set("scope", scope)
	}
	if svmName != "" {
		params.Set("svm.name", svmName)
	}

	builder := c.baseRequestBuilder(`/api/network/ip/interfaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&interfaceData)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if interfaceData.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of network interface name %s because it does not exist", interfaceName)
	}

	if interfaceData.NumRecords > 1 {
		return fmt.Errorf("multiple network interfaces found with name %s; please specify additional filters", interfaceName)
	}

	builder = c.baseRequestBuilder(`/api/network/ip/interfaces/`+interfaceData.Records[0].UUID, &statusCode, responseHeaders).
		BodyJSON(nwInterface).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNetworkIPInterface(ctx context.Context, scope string, interfaceName string, svmName string) error {
	var (
		statusCode    int
		interfaceData ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", interfaceName)
	if scope != "" {
		params.Set("scope", scope)
	}
	if svmName != "" {
		params.Set("svm.name", svmName)
	}

	builder := c.baseRequestBuilder(`/api/network/ip/interfaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&interfaceData)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if interfaceData.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of network interface name %s because it does not exist", interfaceName)
	}

	if interfaceData.NumRecords > 1 {
		return fmt.Errorf("multiple network interfaces found with name %s; please specify additional filters", interfaceName)
	}

	builder = c.baseRequestBuilder(`/api/network/ip/interfaces/`+interfaceData.Records[0].UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
