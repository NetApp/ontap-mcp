package rest

import (
	"context"
	"net/http"

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
	var statusCode int
	responseHeaders := http.Header{}

	svmUUID, err := c.getSVMUUID(ctx, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/name-services/dns/`+svmUUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
