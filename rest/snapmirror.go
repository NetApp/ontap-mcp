package rest

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

// getSnapMirrorUUID returns the UUID of a SnapMirror relationship identified by its destination path.
func (c *Client) getSnapMirrorUUID(ctx context.Context, destSVM, destVolume string) (string, error) {
	var data ontap.GetData

	params := url.Values{}
	params.Set("destination.path", fmt.Sprintf("%s:%s", destSVM, destVolume))
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships`, nil, nil).
		Params(params).
		ToJSON(&data)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if data.NumRecords == 0 {
		return "", fmt.Errorf("SnapMirror relationship with destination %s:%s not found", destSVM, destVolume)
	}
	if data.NumRecords != 1 {
		return "", fmt.Errorf("found %d SnapMirror relationships with destination %s:%s, expected 1", data.NumRecords, destSVM, destVolume)
	}

	return data.Records[0].UUID, nil
}

func (c *Client) CreateSnapMirror(ctx context.Context, rel ontap.SnapMirrorRelationship) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships`, &statusCode, nil).
		BodyJSON(rel).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) UpdateSnapMirror(ctx context.Context, destSVM, destVolume string, rel ontap.SnapMirrorRelationship) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	uuid, err := c.getSnapMirrorUUID(ctx, destSVM, destVolume)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid, &statusCode, nil).
		Patch().
		BodyJSON(rel).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteSnapMirror(ctx context.Context, destSVM, destVolume string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	uuid, err := c.getSnapMirrorUUID(ctx, destSVM, destVolume)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid, &statusCode, nil).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) InitializeSMUpdateSMTransfer(ctx context.Context, destSVM, destVolume string) error {
	var (
		statusCode int
	)

	uuid, err := c.getSnapMirrorUUID(ctx, destSVM, destVolume)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid+`/transfers`, &statusCode, nil).
		BodyJSON(struct{}{})

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) BreakSnapMirror(ctx context.Context, destSVM, destVolume string, rel ontap.SnapMirrorRelationship) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	uuid, err := c.getSnapMirrorUUID(ctx, destSVM, destVolume)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid, &statusCode, nil).
		Patch().
		BodyJSON(rel).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) ResyncSnapMirror(ctx context.Context, destSVM, destVolume string, rel ontap.SnapMirrorRelationship) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	uuid, err := c.getSnapMirrorUUID(ctx, destSVM, destVolume)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid, &statusCode, nil).
		Patch().
		BodyJSON(rel).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}
