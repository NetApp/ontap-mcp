package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateDNS(ctx context.Context, dns ontap.DNSConfig) error {
	var statusCode int
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/name-services/dns`, &statusCode, responseHeaders).
		BodyJSON(dns)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteDNS(ctx context.Context, svmName string) error {
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

	builder2 := c.baseRequestBuilder(`/api/name-services/dns/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
