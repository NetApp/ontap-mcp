package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) GetNFSExportPolicy(ctx context.Context) ([]string, error) {
	var (
		nfsExportPolicy ontap.GetData
	)
	responseHeaders := http.Header{}
	nfsExportPolicies := []string{}

	params := url.Values{}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&nfsExportPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return []string{}, err
	}

	if nfsExportPolicy.NumRecords == 0 {
		return []string{}, errors.New("no nfs export policies found in the cluster")
	}

	for _, nfsExportPolicyData := range nfsExportPolicy.Records {
		nfsExportPolicies = append(nfsExportPolicies, nfsExportPolicyData.Name)
	}

	return nfsExportPolicies, nil
}

func (c *Client) CreateNFSExportPolicy(ctx context.Context, exportPolicy ontap.ExportPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, &statusCode, responseHeaders).
		BodyJSON(exportPolicy).
		ToBytesBuffer(&buf)

	err := c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		return nil
	}
	return err
}

func (c *Client) CreateNFSExportPolicyRules(ctx context.Context, exportPolicyName string, rule ontap.Rule) error {
	var (
		buf          bytes.Buffer
		statusCode   int
		exportPolicy ontap.GetData
	)

	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "id")
	params.Set("name", exportPolicyName)

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&exportPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if exportPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of export policy %s because it does not exist", exportPolicyName)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(exportPolicy.Records[0].ID)+`/rules`, &statusCode, responseHeaders).
		BodyJSON(rule).
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		return nil
	}
	return err
}

func (c *Client) UpdateNFSExportPolicyRules(ctx context.Context, exportPolicyName string, oldClientMatch string, oldRoRule string, oldRwRule string, rule ontap.Rule) error {
	var (
		buf              bytes.Buffer
		statusCode       int
		exportPolicy     ontap.GetData
		exportPolicyRule ontap.GetData
	)

	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "id")
	params.Set("name", exportPolicyName)

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&exportPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if exportPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of export policy %s because it does not exist", exportPolicyName)
	}

	params = url.Values{}
	params.Set("fields", "index")
	if oldClientMatch != "" {
		params.Set("clients.match", oldClientMatch)
	}
	if oldRoRule != "" {
		params.Set("ro_rule", oldRoRule)
	}
	if oldRwRule != "" {
		params.Set("rw_rule", oldRwRule)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(exportPolicy.Records[0].ID)+`/rules`, nil, responseHeaders).
		Params(params).
		ToJSON(&exportPolicyRule)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if err != nil {
		return err
	}

	if exportPolicyRule.NumRecords == 0 {
		return errors.New("failed to get detail of export policy rule because it does not exist")
	}

	builder3 := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(exportPolicy.Records[0].ID)+`/rules/`+strconv.Itoa(exportPolicyRule.Records[0].Index), &statusCode, responseHeaders).
		Patch().
		BodyJSON(rule).
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder3)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}

func (c *Client) DeleteNFSExportPolicyRules(ctx context.Context, exportPolicyName string, rule ontap.Rule) error {
	var (
		buf              bytes.Buffer
		statusCode       int
		exportPolicy     ontap.GetData
		exportPolicyRule ontap.GetData
	)

	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "id")
	params.Set("name", exportPolicyName)

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&exportPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if exportPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of export policy %s because it does not exist", exportPolicyName)
	}

	params = url.Values{}
	params.Set("fields", "index")
	if rule.ClientsStr != "" {
		params.Set("clients.match", rule.ClientsStr)
	}
	if rule.ROruleStr != "" {
		params.Set("ro_rule", rule.ROruleStr)
	}
	if rule.RWruleStr != "" {
		params.Set("rw_rule", rule.RWruleStr)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(exportPolicy.Records[0].ID)+`/rules`, nil, responseHeaders).
		Params(params).
		ToJSON(&exportPolicyRule)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if err != nil {
		return err
	}

	if exportPolicyRule.NumRecords == 0 {
		return errors.New("failed to get detail of export policy rule because it does not exist")
	}

	builder3 := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(exportPolicy.Records[0].ID)+`/rules/`+strconv.Itoa(exportPolicyRule.Records[0].Index), &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder3)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}

func (c *Client) UpdateNFSExportPolicy(ctx context.Context, oldExportPolicyName string, exportPolicy ontap.ExportPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		exPolicy   ontap.GetData
	)

	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "id")
	params.Set("name", oldExportPolicyName)

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&exPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if exPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of export policy %s because it does not exist", oldExportPolicyName)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(exPolicy.Records[0].ID), &statusCode, responseHeaders).
		Patch().
		BodyJSON(exportPolicy).
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}

func (c *Client) DeleteNFSExportPolicy(ctx context.Context, exportPolicy ontap.ExportPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		exPolicy   ontap.GetData
	)

	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "id")
	params.Set("name", exportPolicy.Name)

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&exPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if exPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of export policy %s because it does not exist", exportPolicy.Name)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(exPolicy.Records[0].ID), &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}
