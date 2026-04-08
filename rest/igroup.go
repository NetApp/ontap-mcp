package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/netapp/ontap-mcp/ontap"
)

func (c *Client) CreateIGroup(ctx context.Context, igroup ontap.IGroup) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/san/igroups`, &statusCode, responseHeaders).
		BodyJSON(igroup)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateIGroup(ctx context.Context, igroup ontap.IGroup, igroupName, svmName string) error {
	var (
		statusCode int
		ig         ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", igroupName)
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/igroups`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&ig)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if ig.NumRecords == 0 {
		return fmt.Errorf("failed to update igroup=%s on svm=%s because it does not exist", igroupName, svmName)
	}
	if ig.NumRecords != 1 {
		return fmt.Errorf("failed to update igroup=%s on svm=%s because there are %d matching records", igroupName, svmName, ig.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/igroups/`+ig.Records[0].UUID, &statusCode, responseHeaders).
		Patch().
		BodyJSON(igroup)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteIGroup(ctx context.Context, igroup ontap.IGroup, allowDeleteWhileMapped bool) error {
	var (
		statusCode int
		ig         ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", igroup.Name)
	params.Set("svm.name", igroup.SVM.Name)

	builder := c.baseRequestBuilder(`/api/protocols/san/igroups`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&ig)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if ig.NumRecords == 0 {
		return fmt.Errorf("failed to delete igroup=%s on svm=%s because it does not exist", igroup.Name, igroup.SVM.Name)
	}
	if ig.NumRecords != 1 {
		return fmt.Errorf("failed to delete igroup=%s on svm=%s because there are %d matching records", igroup.Name, igroup.SVM.Name, ig.NumRecords)
	}

	deleteParams := url.Values{}
	deleteParams.Set("allow_delete_while_mapped", strconv.FormatBool(allowDeleteWhileMapped))
	builder = c.baseRequestBuilder(`/api/protocols/san/igroups/`+ig.Records[0].UUID, &statusCode, responseHeaders).
		Params(deleteParams).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) AddIGroupInitiator(ctx context.Context, igroupName, svmName string, initiator ontap.IGroupInitiator) error {
	var (
		statusCode int
		ig         ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", igroupName)
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/igroups`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&ig)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if ig.NumRecords == 0 {
		return fmt.Errorf("failed to add initiator to igroup=%s on svm=%s because the igroup does not exist", igroupName, svmName)
	}
	if ig.NumRecords != 1 {
		return fmt.Errorf("failed to add initiator to igroup=%s on svm=%s because there are %d matching records", igroupName, svmName, ig.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/protocols/san/igroups/`+ig.Records[0].UUID+`/initiators`, &statusCode, responseHeaders).
		BodyJSON(initiator)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) RemoveIGroupInitiator(ctx context.Context, igroupName, svmName string, initiator ontap.IGroupInitiator, allowDeleteWhileMapped bool) error {
	var (
		statusCode int
		ig         ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", igroupName)
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/san/igroups`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&ig)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if ig.NumRecords == 0 {
		return fmt.Errorf("failed to remove initiator from igroup=%s on svm=%s because the igroup does not exist", igroupName, svmName)
	}
	if ig.NumRecords != 1 {
		return fmt.Errorf("failed to remove initiator from igroup=%s on svm=%s because there are %d matching records", igroupName, svmName, ig.NumRecords)
	}

	deleteParams := url.Values{}
	deleteParams.Set("allow_delete_while_mapped", strconv.FormatBool(allowDeleteWhileMapped))
	builder = c.baseRequestBuilder(`/api/protocols/san/igroups/`+ig.Records[0].UUID+`/initiators/`+url.PathEscape(initiator.Name), &statusCode, responseHeaders).
		Params(deleteParams).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
