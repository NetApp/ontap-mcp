package rest

import (
	"context"
	"net/http"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateNFSService(ctx context.Context, nfsService ontap.NFSService) error {
	var statusCode int
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/services`, &statusCode, responseHeaders).
		BodyJSON(nfsService)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNFSService(ctx context.Context, svmName string, nfsService ontap.NFSService) error {
	var statusCode int
	responseHeaders := http.Header{}

	svmUUID, err := c.getSVMUUID(ctx, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/services/`+svmUUID, &statusCode, responseHeaders).
		BodyJSON(nfsService).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNFSService(ctx context.Context, svmName string) error {
	var statusCode int
	responseHeaders := http.Header{}

	svmUUID, err := c.getSVMUUID(ctx, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/services/`+svmUUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
