package rest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) getVolumeUUID(ctx context.Context, volumeName, svmName string) (string, error) {
	var volData ontap.GetData
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", volumeName)
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/storage/volumes`, nil, responseHeaders).
		Params(params).
		ToJSON(&volData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if volData.NumRecords == 0 {
		return "", fmt.Errorf("volume=%s on svm=%s does not exist", volumeName, svmName)
	}
	if volData.NumRecords != 1 {
		return "", fmt.Errorf("found %d volumes matching name=%s on svm=%s", volData.NumRecords, volumeName, svmName)
	}

	return volData.Records[0].UUID, nil
}

func (c *Client) getSnapshotUUID(ctx context.Context, volumeUUID, snapshotName string) (string, error) {
	var snapData ontap.GetData
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", snapshotName)

	builder := c.baseRequestBuilder(`/api/storage/volumes/`+volumeUUID+`/snapshots`, nil, responseHeaders).
		Params(params).
		ToJSON(&snapData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if snapData.NumRecords == 0 {
		return "", fmt.Errorf("snapshot=%s on volume uuid=%s does not exist", snapshotName, volumeUUID)
	}
	if snapData.NumRecords != 1 {
		return "", fmt.Errorf("found %d snapshots matching name=%s on volume uuid=%s", snapData.NumRecords, snapshotName, volumeUUID)
	}

	return snapData.Records[0].UUID, nil
}

func (c *Client) CreateSnapshot(ctx context.Context, snapshot ontap.Snapshot, volumeName, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	volumeUUID, err := c.getVolumeUUID(ctx, volumeName, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/volumes/`+volumeUUID+`/snapshots`, &statusCode, responseHeaders).
		BodyJSON(snapshot).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteSnapshot(ctx context.Context, volumeName, svmName, snapshotName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	volumeUUID, err := c.getVolumeUUID(ctx, volumeName, svmName)
	if err != nil {
		return err
	}

	snapshotUUID, err := c.getSnapshotUUID(ctx, volumeUUID, snapshotName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/volumes/`+volumeUUID+`/snapshots/`+snapshotUUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) RestoreSnapshot(ctx context.Context, volumeName, svmName string, snapshotRestore ontap.SnapshotRestore) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	volumeUUID, err := c.getVolumeUUID(ctx, volumeName, svmName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/storage/volumes/`+volumeUUID, &statusCode, responseHeaders).
		BodyJSON(snapshotRestore).
		Patch().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}
