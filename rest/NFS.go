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

func (c *Client) CreateNFSExportPolicy(ctx context.Context, exportPolicy ontap.ExportPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, &statusCode, responseHeaders).
		BodyJSON(exportPolicy).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

// getNFSExportPolicyID returns the id of the export policy named exportPolicyName on svmName.
// Policy names are only unique within an SVM (every SVM has a "default" policy), so the lookup
// is scoped to the SVM and fails unless exactly one policy matches.
func (c *Client) getNFSExportPolicyID(ctx context.Context, svmName string, exportPolicyName string) (int, error) {
	var exportPolicy ontap.GetData

	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "id")
	params.Set("name", exportPolicyName)
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&exportPolicy)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return 0, err
	}

	if exportPolicy.NumRecords == 0 {
		return 0, fmt.Errorf("failed to get detail of export policy %s because it does not exist on svm %s", exportPolicyName, svmName)
	}
	if exportPolicy.NumRecords > 1 {
		return 0, fmt.Errorf("found %d export policies named %s on svm %s, expected 1", exportPolicy.NumRecords, exportPolicyName, svmName)
	}

	return exportPolicy.Records[0].ID, nil
}

// getNFSExportPolicyRuleIndex returns the index of the rule in export policy policyID that matches
// the given client match, ro rule, and rw rule. Empty selectors are not filtered on. It fails
// unless exactly one rule matches.
func (c *Client) getNFSExportPolicyRuleIndex(ctx context.Context, policyID int, exportPolicyName string, svmName string, clientMatch string, roRule string, rwRule string) (int, error) {
	var exportPolicyRule ontap.GetData

	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "index")
	if clientMatch != "" {
		params.Set("clients.match", clientMatch)
	}
	if roRule != "" {
		params.Set("ro_rule", roRule)
	}
	if rwRule != "" {
		params.Set("rw_rule", rwRule)
	}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(policyID)+`/rules`, nil, responseHeaders).
		Params(params).
		ToJSON(&exportPolicyRule)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return 0, err
	}

	if exportPolicyRule.NumRecords == 0 {
		return 0, errors.New("failed to get detail of export policy rule because it does not exist")
	}
	if exportPolicyRule.NumRecords > 1 {
		return 0, fmt.Errorf("found %d export policy rules matching old_client=%q old_ro_rule=%q old_rw_rule=%q in export policy %s on svm %s, expected 1",
			exportPolicyRule.NumRecords, clientMatch, roRule, rwRule, exportPolicyName, svmName)
	}

	return exportPolicyRule.Records[0].Index, nil
}

func (c *Client) CreateNFSExportPolicyRules(ctx context.Context, svmName string, exportPolicyName string, rule ontap.Rule) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	policyID, err := c.getNFSExportPolicyID(ctx, svmName, exportPolicyName)
	if err != nil {
		return err
	}

	responseHeaders := http.Header{}
	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(policyID)+`/rules`, &statusCode, responseHeaders).
		BodyJSON(rule).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNFSExportPolicyRules(ctx context.Context, svmName string, exportPolicyName string, oldClientMatch string, oldRoRule string, oldRwRule string, rule ontap.Rule) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	policyID, err := c.getNFSExportPolicyID(ctx, svmName, exportPolicyName)
	if err != nil {
		return err
	}

	ruleIndex, err := c.getNFSExportPolicyRuleIndex(ctx, policyID, exportPolicyName, svmName, oldClientMatch, oldRoRule, oldRwRule)
	if err != nil {
		return err
	}

	responseHeaders := http.Header{}
	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(policyID)+`/rules/`+strconv.Itoa(ruleIndex), &statusCode, responseHeaders).
		Patch().
		BodyJSON(rule).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNFSExportPolicyRules(ctx context.Context, svmName string, exportPolicyName string, rule ontap.Rule) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	policyID, err := c.getNFSExportPolicyID(ctx, svmName, exportPolicyName)
	if err != nil {
		return err
	}

	ruleIndex, err := c.getNFSExportPolicyRuleIndex(ctx, policyID, exportPolicyName, svmName, rule.ClientsStr, rule.ROruleStr, rule.RWruleStr)
	if err != nil {
		return err
	}

	responseHeaders := http.Header{}
	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(policyID)+`/rules/`+strconv.Itoa(ruleIndex), &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNFSExportPolicy(ctx context.Context, svmName string, oldExportPolicyName string, exportPolicy ontap.ExportPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	policyID, err := c.getNFSExportPolicyID(ctx, svmName, oldExportPolicyName)
	if err != nil {
		return err
	}

	responseHeaders := http.Header{}
	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(policyID), &statusCode, responseHeaders).
		Patch().
		BodyJSON(exportPolicy).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNFSExportPolicy(ctx context.Context, exportPolicy ontap.ExportPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)

	policyID, err := c.getNFSExportPolicyID(ctx, exportPolicy.SVM.Name, exportPolicy.Name)
	if err != nil {
		return err
	}

	responseHeaders := http.Header{}
	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies/`+strconv.Itoa(policyID), &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
