package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/catalog"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

var builtInSnapMirrorPolicies = map[string]struct{}{
	"asynchronous":            {},
	"automatedfailover":       {},
	"automatedfailoverduplex": {},
	"continuous":              {},
	"dpdefault":               {},
	"mirrorallsnapshots":      {},
	"mirrorandvault":          {},
	"mirrorlatest":            {},
	"strictsync":              {},
	"sync":                    {},
	"vault":                   {},
	"xdpdefault":              {},
}

func isBuiltInSnapMirrorPolicy(name string) bool {
	_, ok := builtInSnapMirrorPolicies[strings.ToLower(name)]
	return ok
}

func (a *App) CreateSnapMirrorPolicy(ctx context.Context, _ *mcp.CallToolRequest, p tool.SnapMirrorPolicyCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(p.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", p.Cluster)), nil, nil
	}
	defer a.locks.Unlock(p.Cluster)

	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, nil, errors.New("SnapMirror policy name is required")
	}
	remote, err := a.snapMirrorPolicyRemote(ctx, p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	policy, err := newSnapMirrorPolicy(p, remote)
	if err != nil {
		return nil, nil, err
	}
	client, err := a.getClient(p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	if err := client.CreateSnapMirrorPolicy(ctx, policy); err != nil {
		return errorResult(err), nil, err
	}
	return a.snapMirrorPolicyResult(ctx, client, p.Cluster, name, "created", snapMirrorPolicyHasAsyncFlags(remote))
}

func (a *App) UpdateSnapMirrorPolicy(ctx context.Context, _ *mcp.CallToolRequest, p tool.SnapMirrorPolicyModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(p.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", p.Cluster)), nil, nil
	}
	defer a.locks.Unlock(p.Cluster)

	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, nil, errors.New("SnapMirror policy name is required")
	}
	if isBuiltInSnapMirrorPolicy(name) {
		return errorResult(fmt.Errorf("SnapMirror policy %q is built-in and cannot be modified; create a custom policy instead", name)), nil, nil
	}
	if p.Update.Comment == nil && p.Update.NetworkCompression == nil && p.Update.Throttle == nil {
		return errorResult(errors.New("at least one SnapMirror policy field must be provided")), nil, nil
	}
	if p.Update.Throttle != nil && *p.Update.Throttle < 0 {
		return nil, nil, errors.New("SnapMirror policy throttle cannot be negative")
	}
	remote, err := a.snapMirrorPolicyRemote(ctx, p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	update := ontap.SnapMirrorPolicyConfig{
		Comment:            p.Update.Comment,
		NetworkCompression: p.Update.NetworkCompression,
		Throttle:           p.Update.Throttle,
	}
	client, err := a.getClient(p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	includeAsyncFlags := snapMirrorPolicyHasAsyncFlags(remote)
	if err := client.UpdateSnapMirrorPolicy(ctx, name, update, includeAsyncFlags); err != nil {
		return errorResult(err), nil, err
	}
	return a.snapMirrorPolicyResult(ctx, client, p.Cluster, name, "updated", includeAsyncFlags)
}

func (a *App) DeleteSnapMirrorPolicy(ctx context.Context, _ *mcp.CallToolRequest, p tool.SnapMirrorPolicy) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(p.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", p.Cluster)), nil, nil
	}
	defer a.locks.Unlock(p.Cluster)

	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, nil, errors.New("SnapMirror policy name is required")
	}
	if isBuiltInSnapMirrorPolicy(name) {
		return errorResult(fmt.Errorf("SnapMirror policy %q is built-in and cannot be deleted; create a custom policy instead", name)), nil, nil
	}
	remote, err := a.snapMirrorPolicyRemote(ctx, p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	client, err := a.getClient(p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	if err := client.DeleteSnapMirrorPolicy(ctx, name, snapMirrorPolicyHasAsyncFlags(remote)); err != nil {
		return errorResult(err), nil, err
	}
	return textResult(fmt.Sprintf("SnapMirror policy %q deleted successfully", name)), nil, nil
}

func (a *App) AddRuleInSnapMirrorPolicy(ctx context.Context, _ *mcp.CallToolRequest, p tool.SnapMirrorPolicyRuleCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(p.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", p.Cluster)), nil, nil
	}
	defer a.locks.Unlock(p.Cluster)

	p.Name = strings.TrimSpace(p.Name)
	p.Label = strings.TrimSpace(p.Label)
	if p.Name == "" {
		return nil, nil, errors.New("SnapMirror policy name is required")
	}
	if isBuiltInSnapMirrorPolicy(p.Name) {
		return errorResult(fmt.Errorf("SnapMirror policy %q is built-in and cannot be modified; create a custom policy instead", p.Name)), nil, nil
	}
	remote, err := a.snapMirrorPolicyRemote(ctx, p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	rule := ontap.SnapMirrorPolicyRetention{Label: p.Label, Count: p.Keep, Preserve: p.Preserve, Warn: p.Warn}
	if err := validateSnapMirrorPolicyRule(rule, remote); err != nil {
		return errorResult(err), nil, nil
	}
	client, err := a.getClient(p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	includeAsyncFlags := snapMirrorPolicyHasAsyncFlags(remote)
	if err := client.AddSnapMirrorPolicyRule(ctx, p.Name, rule, includeAsyncFlags); err != nil {
		return errorResult(err), nil, err
	}
	return a.snapMirrorPolicyResult(ctx, client, p.Cluster, p.Name, "rule_added", includeAsyncFlags)
}

func (a *App) UpdateRuleInSnapMirrorPolicy(ctx context.Context, _ *mcp.CallToolRequest, p tool.SnapMirrorPolicyRuleUpdate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(p.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", p.Cluster)), nil, nil
	}
	defer a.locks.Unlock(p.Cluster)

	p.Name = strings.TrimSpace(p.Name)
	p.Label = strings.TrimSpace(p.Label)
	if p.Name == "" {
		return nil, nil, errors.New("SnapMirror policy name is required")
	}
	if isBuiltInSnapMirrorPolicy(p.Name) {
		return errorResult(fmt.Errorf("SnapMirror policy %q is built-in and cannot be modified; create a custom policy instead", p.Name)), nil, nil
	}
	if p.Keep == nil && p.Preserve == nil && p.Warn == nil {
		return errorResult(errors.New("at least one retention rule field must be provided")), nil, nil
	}
	update := ontap.SnapMirrorPolicyRetention{Label: p.Label, Preserve: p.Preserve, Warn: p.Warn}
	if p.Keep != nil {
		if *p.Keep <= 0 {
			return nil, nil, errors.New("retention rule keep count must be greater than zero")
		}
		update.Count = *p.Keep
	}
	remote, err := a.snapMirrorPolicyRemote(ctx, p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	if err := validateSnapMirrorPolicyRuleOptions(update, remote, false); err != nil {
		return errorResult(err), nil, nil
	}
	client, err := a.getClient(p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	includeAsyncFlags := snapMirrorPolicyHasAsyncFlags(remote)
	if err := client.UpdateSnapMirrorPolicyRule(ctx, p.Name, p.Label, update, includeAsyncFlags); err != nil {
		return errorResult(err), nil, err
	}
	return a.snapMirrorPolicyResult(ctx, client, p.Cluster, p.Name, "rule_updated", includeAsyncFlags)
}

func (a *App) RemoveRuleInSnapMirrorPolicy(ctx context.Context, _ *mcp.CallToolRequest, p tool.SnapMirrorPolicyRuleRemove) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(p.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", p.Cluster)), nil, nil
	}
	defer a.locks.Unlock(p.Cluster)

	p.Name = strings.TrimSpace(p.Name)
	p.Label = strings.TrimSpace(p.Label)
	if p.Name == "" {
		return nil, nil, errors.New("SnapMirror policy name is required")
	}
	if isBuiltInSnapMirrorPolicy(p.Name) {
		return errorResult(fmt.Errorf("SnapMirror policy %q is built-in and cannot be modified; create a custom policy instead", p.Name)), nil, nil
	}
	remote, err := a.snapMirrorPolicyRemote(ctx, p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	client, err := a.getClient(p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	includeAsyncFlags := snapMirrorPolicyHasAsyncFlags(remote)
	if err := client.RemoveSnapMirrorPolicyRule(ctx, p.Name, p.Label, includeAsyncFlags); err != nil {
		return errorResult(err), nil, err
	}
	return a.snapMirrorPolicyResult(ctx, client, p.Cluster, p.Name, "rule_removed", includeAsyncFlags)
}

func (a *App) snapMirrorPolicyRemote(ctx context.Context, cluster string) (ontap.Remote, error) {
	remote, err := a.getClusterRemote(ctx, cluster)
	if err != nil {
		return ontap.Remote{}, err
	}
	version := fmt.Sprintf("%d.%d", remote.Version.Generation, remote.Version.Major)
	if catalog.CompareVersions(version, "9.6") < 0 {
		return ontap.Remote{}, fmt.Errorf("SnapMirror policy REST tools require ONTAP 9.6 or later; cluster %q is running %s", cluster, version)
	}
	return remote, nil
}

func snapMirrorPolicyHasAsyncFlags(remote ontap.Remote) bool {
	version := fmt.Sprintf("%d.%d", remote.Version.Generation, remote.Version.Major)
	return catalog.CompareVersions(version, "9.11") >= 0
}

func validateSnapMirrorPolicyRule(rule ontap.SnapMirrorPolicyRetention, remote ontap.Remote) error {
	return validateSnapMirrorPolicyRuleOptions(rule, remote, true)
}

func validateSnapMirrorPolicyRuleOptions(rule ontap.SnapMirrorPolicyRetention, remote ontap.Remote, requireCount bool) error {
	if strings.TrimSpace(rule.Label) == "" {
		return errors.New("SnapMirror retention rule label is required")
	}
	if requireCount && rule.Count <= 0 {
		return errors.New("retention rule keep count must be greater than zero")
	}
	if (rule.Preserve != nil || rule.Warn != nil) && remote.Version.Generation == 9 && remote.Version.Major < 13 {
		return fmt.Errorf("ONTAP %d.%d does not support SnapMirror retention rule preserve or warn settings; these require ONTAP 9.13 or later",
			remote.Version.Generation, remote.Version.Major)
	}
	if rule.Warn != nil && *rule.Warn < 0 {
		return errors.New("retention rule warn threshold cannot be negative")
	}
	return nil
}

func newSnapMirrorPolicy(p tool.SnapMirrorPolicyCreate, remote ontap.Remote) (ontap.SnapMirrorPolicyConfig, error) {
	if p.Throttle != nil && *p.Throttle < 0 {
		return ontap.SnapMirrorPolicyConfig{}, errors.New("SnapMirror policy throttle cannot be negative")
	}
	policyType := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(p.Type)))
	switch policyType {
	case "async", "mirror_vault":
	case "vault":
		if !snapMirrorPolicyHasAsyncFlags(remote) {
			return ontap.SnapMirrorPolicyConfig{}, fmt.Errorf("ONTAP %d.%d does not support vault SnapMirror policies; these require ONTAP 9.11 or later",
				remote.Version.Generation, remote.Version.Major)
		}
	case "async_mirror":
		if !snapMirrorPolicyHasAsyncFlags(remote) {
			return ontap.SnapMirrorPolicyConfig{}, fmt.Errorf("ONTAP %d.%d does not support async_mirror SnapMirror policies; these require ONTAP 9.11 or later",
				remote.Version.Generation, remote.Version.Major)
		}
		if len(p.RetentionRules) != 0 {
			return ontap.SnapMirrorPolicyConfig{}, errors.New("async_mirror policies use copy_latest_source_snapshot and cannot define retention rules; use mirror_vault for custom retention")
		}
	default:
		return ontap.SnapMirrorPolicyConfig{}, fmt.Errorf("unsupported SnapMirror policy type %q; supported values: async, async_mirror, vault, mirror_vault", p.Type)
	}
	policy := ontap.SnapMirrorPolicyConfig{
		Name:               strings.TrimSpace(p.Name),
		Type:               "async",
		NetworkCompression: p.NetworkCompression,
		Throttle:           p.Throttle,
	}
	if p.Comment != "" {
		comment := p.Comment
		policy.Comment = &comment
	}
	if policyType == "vault" {
		no := false
		policy.CreateSnapshotOnSource = &no
	}
	if policyType == "async_mirror" {
		yes := true
		policy.CopyLatestSourceSnapshot = &yes
	}
	for _, rule := range p.RetentionRules {
		converted := ontap.SnapMirrorPolicyRetention{
			Label:    rule.Label,
			Count:    rule.Keep,
			Preserve: rule.Preserve,
			Warn:     rule.Warn,
		}
		if err := validateSnapMirrorPolicyRule(converted, remote); err != nil {
			return ontap.SnapMirrorPolicyConfig{}, err
		}
		for _, existing := range policy.Retention {
			if existing.Label == converted.Label {
				return ontap.SnapMirrorPolicyConfig{}, fmt.Errorf("duplicate SnapMirror retention rule label %q", converted.Label)
			}
		}
		policy.Retention = append(policy.Retention, converted)
	}
	if strings.EqualFold(strings.TrimSpace(p.Type), "vault") && len(policy.Retention) == 0 {
		return ontap.SnapMirrorPolicyConfig{}, errors.New("vault SnapMirror policies require at least one retention rule")
	}
	return policy, nil
}

func (a *App) snapMirrorPolicyResult(ctx context.Context, client interface {
	GetSnapMirrorPolicy(context.Context, string, bool) (ontap.SnapMirrorPolicyConfig, error)
	SnapMirrorPolicyLabelProduced(context.Context, string) (bool, error)
}, cluster, name, action string, includeAsyncFlags bool) (*mcp.CallToolResult, any, error) {
	policy, err := client.GetSnapMirrorPolicy(ctx, name, includeAsyncFlags)
	if err != nil {
		return nil, nil, fmt.Errorf("SnapMirror policy %q was changed successfully, but failed to retrieve its current state: %w", name, err)
	}

	var b strings.Builder
	switch action {
	case "created", "updated":
		fmt.Fprintf(&b, "SnapMirror policy %q %s successfully.\n", name, action)
	case "rule_added":
		fmt.Fprintf(&b, "Retention rule added to SnapMirror policy %q successfully.\n", name)
	case "rule_updated":
		fmt.Fprintf(&b, "Retention rule updated in SnapMirror policy %q successfully.\n", name)
	case "rule_removed":
		fmt.Fprintf(&b, "Retention rule removed from SnapMirror policy %q successfully.\n", name)
	}
	fmt.Fprintf(&b, "Current policy: type=%s", policy.Type)
	if policy.Comment != nil && *policy.Comment != "" {
		fmt.Fprintf(&b, ", comment=%q", *policy.Comment)
	}
	if policy.NetworkCompression != nil {
		fmt.Fprintf(&b, ", network_compression_enabled=%t", *policy.NetworkCompression)
	}
	if policy.Throttle != nil {
		fmt.Fprintf(&b, ", throttle=%d KB/s", *policy.Throttle)
	}
	if policy.CreateSnapshotOnSource != nil {
		fmt.Fprintf(&b, ", create_snapshot_on_source=%t", *policy.CreateSnapshotOnSource)
	}
	if policy.CopyLatestSourceSnapshot != nil {
		fmt.Fprintf(&b, ", copy_latest_source_snapshot=%t", *policy.CopyLatestSourceSnapshot)
	}
	b.WriteString("\nRetention rules:\n")
	if len(policy.Retention) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, rule := range policy.Retention {
			fmt.Fprintf(&b, "- label=%q, keep=%d", rule.Label, rule.Count)
			if rule.Preserve != nil {
				fmt.Fprintf(&b, ", preserve=%t", *rule.Preserve)
			}
			if rule.Warn != nil {
				fmt.Fprintf(&b, ", warn=%d", *rule.Warn)
			}
			produced, err := client.SnapMirrorPolicyLabelProduced(ctx, rule.Label)
			if err != nil {
				return nil, nil, fmt.Errorf("SnapMirror policy %q was changed successfully, but snapshot label %q could not be verified: %w", name, rule.Label, err)
			}
			if produced {
				b.WriteString("; snapshot schedule label: matched\n")
			} else {
				b.WriteString("; WARNING: no snapshot policy schedule entry produces this snapmirror_label; this rule retains nothing until a matching entry is configured\n")
			}
		}
	}
	fmt.Fprintf(&b, "Target cluster %q accepts transfer attributes: network_compression_enabled, throttle.", cluster)
	return textResult(b.String()), nil, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
