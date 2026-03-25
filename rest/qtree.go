package rest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) CreateQtree(ctx context.Context, qtree ontap.Qtree) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/storage/qtrees`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		BodyJSON(qtree)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) UpdateQtree(ctx context.Context, svmName, volumeName, qtreeName string, qtree ontap.Qtree) error {
	var (
		buf         bytes.Buffer
		statusCode  int
		qtreeRecord ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", qtreeName)
	params.Set("svm", svmName)
	params.Set("volume", volumeName)
	params.Set("fields", "id,volume.uuid")

	builder := c.baseRequestBuilder(`/api/storage/qtrees`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&qtreeRecord)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if qtreeRecord.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of qtree %s because it does not exist", qtreeName)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/qtrees/`+qtreeRecord.Records[0].Volume.UUID+`/`+strconv.Itoa(qtreeRecord.Records[0].ID), &statusCode, responseHeaders).
		BodyJSON(qtree).
		ToBytesBuffer(&buf).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteQtree(ctx context.Context, qtree ontap.Qtree) error {
	var (
		buf         bytes.Buffer
		statusCode  int
		qtreeRecord ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", qtree.Name)
	params.Set("svm", qtree.SVM.Name)
	params.Set("volume", qtree.Volume.Name)
	params.Set("fields", "id,volume.uuid")

	builder := c.baseRequestBuilder(`/api/storage/qtrees`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&qtreeRecord)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if qtreeRecord.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of qtree %s because it does not exist", qtree.Name)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/qtrees/`+qtreeRecord.Records[0].Volume.UUID+`/`+strconv.Itoa(qtreeRecord.Records[0].ID), &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}
