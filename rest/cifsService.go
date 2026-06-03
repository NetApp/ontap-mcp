package rest

import (
	"bytes"
	"context"
	"net/http"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateCIFSService(ctx context.Context, cifsService ontap.CIFSServiceBody) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/cifs/services`, &statusCode, responseHeaders).
		BodyJSON(cifsService).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) UpdateCIFSService(ctx context.Context, svmName string, cifsService ontap.CIFSServiceBody) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	svmUUID, err := c.getSVMUUID(ctx, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/protocols/cifs/services/`+svmUUID, &statusCode, responseHeaders).
		BodyJSON(cifsService).
		ToBytesBuffer(&buf).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteCIFSService(ctx context.Context, svmName, adUser, adPassword string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	svmUUID, err := c.getSVMUUID(ctx, svmName)
	if err != nil {
		return err
	}

	type deleteBody struct {
		ADDomain *ontap.CIFSServiceADDomain `json:"ad_domain,omitempty"`
	}
	var body *deleteBody
	if adUser != "" && adPassword != "" {
		body = &deleteBody{
			ADDomain: &ontap.CIFSServiceADDomain{
				User:     adUser,
				Password: adPassword,
			},
		}
	}

	builder := c.baseRequestBuilder(`/api/protocols/cifs/services/`+svmUUID, &statusCode, responseHeaders).
		ToBytesBuffer(&buf)

	if body != nil {
		builder = builder.BodyJSON(body)
	}

	builder = builder.Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}
