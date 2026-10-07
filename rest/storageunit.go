package rest

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateStorageUnit(ctx context.Context, storageUnit ontap.StorageUnit) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	builder := c.baseRequestBuilder(`/api/storage/storage-units`, &statusCode, http.Header{}).
		BodyJSON(storageUnit).
		ToBytesBuffer(&buf)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) UpdateStorageUnit(ctx context.Context, storageUnit ontap.StorageUnit, name, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	uuid, err := c.getStorageUnitUUID(ctx, name, svmName)
	if err != nil {
		return err
	}
	builder := c.baseRequestBuilder(`/api/storage/storage-units/`+uuid, &statusCode, http.Header{}).
		Patch().
		BodyJSON(storageUnit).
		ToBytesBuffer(&buf)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) DeleteStorageUnit(ctx context.Context, name, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	uuid, err := c.getStorageUnitUUID(ctx, name, svmName)
	if err != nil {
		return err
	}
	builder := c.baseRequestBuilder(`/api/storage/storage-units/`+uuid, &statusCode, http.Header{}).
		Delete().
		ToBytesBuffer(&buf)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) getStorageUnitUUID(ctx context.Context, name, svmName string) (string, error) {
	var data ontap.GetData
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", name)
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/storage/storage-units`, nil, nil).
		Params(params).
		ToJSON(&data)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if data.NumRecords == 0 {
		return "", fmt.Errorf("storage unit %q on SVM %q does not exist", name, svmName)
	}
	if data.NumRecords != 1 {
		return "", fmt.Errorf("found %d storage units named %q on SVM %q", data.NumRecords, name, svmName)
	}
	return data.Records[0].UUID, nil
}
