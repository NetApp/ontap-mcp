package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

// GetSnapMirrorUUIDAndType returns the UUID and Policy type of SnapMirror relationship identified by its destination path.
func (c *Client) GetSnapMirrorUUIDAndType(ctx context.Context, destPath string) (string, string, error) {
	var data ontap.GetData

	params := url.Values{}
	params.Set("destination.path", destPath)
	params.Set("fields", "uuid,policy.type")

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships`, nil, nil).
		Params(params).
		ToJSON(&data)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", "", err
	}

	if data.NumRecords == 0 {
		return "", "", fmt.Errorf("SnapMirror relationship with destination %q not found", destPath)
	}
	if data.NumRecords != 1 {
		return "", "", fmt.Errorf("found %d SnapMirror relationships with destination %q, expected 1", data.NumRecords, destPath)
	}

	return data.Records[0].UUID, data.Records[0].Policy.Type, nil
}

// getSnapMirrorTransferUUID returns the UUID of an in-progress transfer (state=transferring) and queued transfer (state=queued) for the given SnapMirror relationship UUID.
func (c *Client) getSnapMirrorTransferUUID(ctx context.Context, uuid string) ([]string, error) {
	var data ontap.GetData
	var UUIDs []string

	params := url.Values{}
	params.Set("state", "transferring|queued")
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid+`/transfers`, nil, nil).
		Params(params).
		ToJSON(&data)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return []string{}, err
	}

	if data.NumRecords == 0 {
		return []string{}, errors.New("SnapMirror transfer with state transferring or queued not found")
	}

	for _, record := range data.Records {
		UUIDs = append(UUIDs, record.UUID)
	}

	return UUIDs, nil
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

	return c.handleJob(ctx, statusCode, &buf)
}

// UpdateSnapMirror updates the SnapMirror relationship identified by its UUID.
func (c *Client) UpdateSnapMirror(ctx context.Context, uuid string, rel ontap.SnapMirrorRelationship) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid, &statusCode, nil).
		Patch().
		BodyJSON(rel).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) DeleteSnapMirror(ctx context.Context, destPath string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	uuid, _, err := c.GetSnapMirrorUUIDAndType(ctx, destPath)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid, &statusCode, nil).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) UpdateSnapMirrorTransfer(ctx context.Context, destPath string) error {
	var (
		statusCode int
	)

	uuid, _, err := c.GetSnapMirrorUUIDAndType(ctx, destPath)
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

func (c *Client) AbortSnapMirrorTransfer(ctx context.Context, destPath string, rel ontap.SnapMirrorTransfer) error {
	var (
		buf        bytes.Buffer
		errs       []error
		statusCode int
	)

	uuid, _, err := c.GetSnapMirrorUUIDAndType(ctx, destPath)
	if err != nil {
		return err
	}

	transferUUIDs, err := c.getSnapMirrorTransferUUID(ctx, uuid)
	if err != nil {
		return err
	}

	for _, transferUUID := range transferUUIDs {
		builder := c.baseRequestBuilder(`/api/snapmirror/relationships/`+uuid+`/transfers/`+transferUUID, &statusCode, nil).
			Patch().
			BodyJSON(rel).
			ToBytesBuffer(&buf)

		if e := c.buildAndExecuteRequest(ctx, builder); e != nil {
			errs = append(errs, e)
		}

		if e := c.handleJob(ctx, statusCode, &buf); e != nil {
			errs = append(errs, e)
		}
	}
	return errors.Join(errs...)
}
