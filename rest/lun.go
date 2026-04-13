package rest

import (
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) CreateLUN(ctx context.Context, lun ontap.LUN) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/storage/luns`, &statusCode, responseHeaders).
		BodyJSON(lun)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateLUN(ctx context.Context, svmName, lunPath string, lun ontap.LUN) error {
	var (
		statusCode int
		lunData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", lunPath)
	params.Set("svm.name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/storage/luns`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&lunData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if lunData.NumRecords == 0 {
		return fmt.Errorf("failed to get lun=%s on svm=%s because it does not exist", lunPath, svmName)
	}
	if lunData.NumRecords != 1 {
		return fmt.Errorf("failed to get lun=%s on svm=%s because there are %d matching records", lunPath, svmName, lunData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/luns/`+lunData.Records[0].UUID, &statusCode, responseHeaders).
		Patch().
		BodyJSON(lun)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteLUN(ctx context.Context, svmName, lunPath string, allowDeleteWhileMapped bool) error {
	var (
		statusCode int
		lunData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", lunPath)
	params.Set("svm.name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/storage/luns`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&lunData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if lunData.NumRecords == 0 {
		return fmt.Errorf("failed to get lun=%s on svm=%s because it does not exist", lunPath, svmName)
	}
	if lunData.NumRecords != 1 {
		return fmt.Errorf("failed to get lun=%s on svm=%s because there are %d matching records", lunPath, svmName, lunData.NumRecords)
	}

	deleteParams := url.Values{}
	deleteParams.Set("allow_delete_while_mapped", strconv.FormatBool(allowDeleteWhileMapped))
	builder2 := c.baseRequestBuilder(`/api/storage/luns/`+lunData.Records[0].UUID, &statusCode, responseHeaders).
		Params(deleteParams).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
