package rest

import (
	"bytes"
	"context"
	"net/http"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateDNS(ctx context.Context, dns ontap.DNSConfig) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/name-services/dns`, &statusCode, responseHeaders).
		BodyJSON(dns).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteDNS(ctx context.Context, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	svmUUID, err := c.getSVMUUID(ctx, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/name-services/dns/`+svmUUID, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}
