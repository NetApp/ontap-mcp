package rest

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/netapp/ontap-mcp/ontap"
)

const snapMirrorPolicyFields = "uuid,name,type,comment,retention,network_compression_enabled,throttle"

func (c *Client) GetSnapMirrorPolicy(ctx context.Context, name string, includeAsyncFlags bool) (ontap.SnapMirrorPolicyConfig, error) {
	var data ontap.SnapMirrorPolicyCollection
	params := url.Values{}
	fields := snapMirrorPolicyFields
	if includeAsyncFlags {
		fields += ",create_snapshot_on_source,copy_latest_source_snapshot"
	}
	params.Set("fields", fields)
	params.Set("name", name)
	params.Set("scope", "cluster")

	builder := c.baseRequestBuilder("/api/snapmirror/policies", nil, nil).
		Params(params).
		ToJSON(&data)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return ontap.SnapMirrorPolicyConfig{}, err
	}
	if data.NumRecords == 0 {
		return ontap.SnapMirrorPolicyConfig{}, fmt.Errorf("SnapMirror policy %q not found", name)
	}
	if data.NumRecords != 1 {
		return ontap.SnapMirrorPolicyConfig{}, fmt.Errorf("found %d SnapMirror policies named %q, expected 1", data.NumRecords, name)
	}
	return data.Records[0], nil
}

func (c *Client) CreateSnapMirrorPolicy(ctx context.Context, policy ontap.SnapMirrorPolicyConfig) error {
	var statusCode int
	builder := c.baseRequestBuilder("/api/snapmirror/policies", &statusCode, nil).
		BodyJSON(policy)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}
	return c.checkStatus(statusCode)
}

func (c *Client) UpdateSnapMirrorPolicy(ctx context.Context, name string, update ontap.SnapMirrorPolicyConfig, includeAsyncFlags bool) error {
	policy, err := c.GetSnapMirrorPolicy(ctx, name, includeAsyncFlags)
	if err != nil {
		return err
	}
	var statusCode int
	builder := c.baseRequestBuilder("/api/snapmirror/policies/"+policy.UUID, &statusCode, nil).
		Patch().
		BodyJSON(update)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}
	return c.checkStatus(statusCode)
}

func (c *Client) DeleteSnapMirrorPolicy(ctx context.Context, name string, includeAsyncFlags bool) error {
	policy, err := c.GetSnapMirrorPolicy(ctx, name, includeAsyncFlags)
	if err != nil {
		return err
	}

	var relationships ontap.SnapMirrorRelationshipCollection
	params := url.Values{}
	params.Set("fields", "source.path,destination.path")
	params.Set("max_records", "10000")
	params.Set("policy.name", name)
	builder := c.baseRequestBuilder("/api/snapmirror/relationships", nil, nil).
		Params(params).
		ToJSON(&relationships)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}
	if relationships.NumRecords > 0 {
		blockers := make([]string, 0, len(relationships.Records))
		for _, rel := range relationships.Records {
			blockers = append(blockers, fmt.Sprintf("%s -> %s", rel.Source.Path, rel.Destination.Path))
		}
		return fmt.Errorf("cannot delete SnapMirror policy %q because it is in use by relationships: %s; reassign or delete those relationships first",
			name, strings.Join(blockers, ", "))
	}

	var statusCode int
	builder = c.baseRequestBuilder("/api/snapmirror/policies/"+policy.UUID, &statusCode, nil).Delete()
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}
	return c.checkStatus(statusCode)
}

func (c *Client) AddSnapMirrorPolicyRule(ctx context.Context, name string, rule ontap.SnapMirrorPolicyRetention, includeAsyncFlags bool) error {
	policy, err := c.GetSnapMirrorPolicy(ctx, name, includeAsyncFlags)
	if err != nil {
		return err
	}
	if policy.CopyLatestSourceSnapshot != nil && *policy.CopyLatestSourceSnapshot {
		return fmt.Errorf("SnapMirror policy %q uses copy_latest_source_snapshot and cannot have custom retention rules; use a mirror_vault policy instead", name)
	}
	for _, current := range policy.Retention {
		if current.Label == rule.Label {
			return fmt.Errorf("retention rule label %q already exists in SnapMirror policy %q", rule.Label, name)
		}
	}
	policy.Retention = append(policy.Retention, rule)
	return c.patchSnapMirrorPolicyRetention(ctx, policy)
}

func (c *Client) UpdateSnapMirrorPolicyRule(ctx context.Context, name, label string, update ontap.SnapMirrorPolicyRetention, includeAsyncFlags bool) error {
	policy, err := c.GetSnapMirrorPolicy(ctx, name, includeAsyncFlags)
	if err != nil {
		return err
	}
	found := false
	for i := range policy.Retention {
		if policy.Retention[i].Label != label {
			continue
		}
		found = true
		if update.Count != 0 {
			policy.Retention[i].Count = update.Count
		}
		if update.Preserve != nil {
			policy.Retention[i].Preserve = update.Preserve
		}
		if update.Warn != nil {
			policy.Retention[i].Warn = update.Warn
		}
		break
	}
	if !found {
		return fmt.Errorf("retention rule label %q does not exist in SnapMirror policy %q", label, name)
	}
	return c.patchSnapMirrorPolicyRetention(ctx, policy)
}

func (c *Client) RemoveSnapMirrorPolicyRule(ctx context.Context, name, label string, includeAsyncFlags bool) error {
	policy, err := c.GetSnapMirrorPolicy(ctx, name, includeAsyncFlags)
	if err != nil {
		return err
	}
	rules := make([]ontap.SnapMirrorPolicyRetention, 0, len(policy.Retention))
	found := false
	for _, rule := range policy.Retention {
		if rule.Label == label {
			found = true
			continue
		}
		rules = append(rules, rule)
	}
	if !found {
		return fmt.Errorf("retention rule label %q does not exist in SnapMirror policy %q", label, name)
	}
	policy.Retention = rules
	return c.patchSnapMirrorPolicyRetention(ctx, policy)
}

func (c *Client) patchSnapMirrorPolicyRetention(ctx context.Context, policy ontap.SnapMirrorPolicyConfig) error {
	var statusCode int
	update := struct {
		Retention []ontap.SnapMirrorPolicyRetention `json:"retention"`
	}{Retention: policy.Retention}
	builder := c.baseRequestBuilder("/api/snapmirror/policies/"+policy.UUID, &statusCode, nil).
		Patch().
		BodyJSON(update)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}
	return c.checkStatus(statusCode)
}

func (c *Client) SnapMirrorPolicyLabelProduced(ctx context.Context, label string) (bool, error) {
	var data ontap.OnlyCount
	params := url.Values{}
	params.Set("copies.snapmirror_label", label)
	params.Set("fields", "uuid")
	params.Set("return_records", "false")
	builder := c.baseRequestBuilder("/api/storage/snapshot-policies", nil, nil).
		Params(params).
		ToJSON(&data)
	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return false, err
	}
	return data.NumRecords > 0, nil
}
