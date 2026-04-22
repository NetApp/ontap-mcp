package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateLunMap(ctx context.Context, lunMap ontap.LunMap) error {
	var statusCode int
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/san/lun-maps`, &statusCode, responseHeaders).
		BodyJSON(lunMap)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteLunMap(ctx context.Context, svmName, lunName, igroupName string) error {
	var (
		statusCode int
		lm         ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("fields", "lun.uuid,igroup.uuid")
	params.Set("svm.name", svmName)
	params.Set("lun.name", lunName)
	params.Set("igroup.name", igroupName)

	builder := c.baseRequestBuilder(`/api/protocols/san/lun-maps`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&lm)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if lm.NumRecords == 0 {
		return fmt.Errorf("failed to find lun map for lun=%s igroup=%s on svm=%s because it does not exist", lunName, igroupName, svmName)
	}
	if lm.NumRecords != 1 {
		return fmt.Errorf("failed to find lun map for lun=%s igroup=%s on svm=%s because there are %d matching records", lunName, igroupName, svmName, lm.NumRecords)
	}

	lunUUID := lm.Records[0].Lun.UUID
	igroupUUID := lm.Records[0].IGroup.UUID

	builder = c.baseRequestBuilder(`/api/protocols/san/lun-maps/`+url.PathEscape(lunUUID)+`/`+url.PathEscape(igroupUUID), &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
