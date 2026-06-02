package rest

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

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
		svmData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if svmData.NumRecords == 0 {
		return fmt.Errorf("failed to get details of SVM %s because it does not exist", svmName)
	}
	if svmData.NumRecords != 1 {
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records", svmName, svmData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/cifs/services/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		BodyJSON(cifsService).
		ToBytesBuffer(&buf).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteCIFSService(ctx context.Context, svmName, adUser, adPassword string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		svmData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if svmData.NumRecords == 0 {
		return fmt.Errorf("failed to get details of SVM %s because it does not exist", svmName)
	}
	if svmData.NumRecords != 1 {
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records", svmName, svmData.NumRecords)
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

	builder2 := c.baseRequestBuilder(`/api/protocols/cifs/services/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		Delete()

	if body != nil {
		builder2 = builder2.BodyJSON(body)
	}

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}
