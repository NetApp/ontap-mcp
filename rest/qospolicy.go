package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) GetAdminSVM(ctx context.Context) (string, error) {
	var result struct {
		Records []struct {
			Vserver string `json:"vserver"`
		} `json:"records"`
	}
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("type", "admin")
	params.Set("fields", "vserver")
	params.Set("max_records", "1")

	builder := c.baseRequestBuilder(`/api/private/cli/vserver`, nil, responseHeaders).
		Params(params).
		ToJSON(&result)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}
	if len(result.Records) == 0 {
		return "", errors.New("admin vserver not found")
	}
	return result.Records[0].Vserver, nil
}

func (c *Client) GetSVMQoSPolicies(ctx context.Context, svmName string) ([]json.RawMessage, error) {
	params := url.Values{}
	params.Set("fields", "*")
	if svmName != "" {
		params.Set("svm.name", svmName)
	}

	raw, err := c.GenericGet(ctx, "/storage/qos/policies", params, 0)
	if err != nil {
		return nil, err
	}
	var result struct {
		Records []json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}

func (c *Client) GetAdminSVMFixedPolicies(ctx context.Context, adminVserver string) ([]json.RawMessage, error) {
	params := url.Values{}
	params.Set("vserver", adminVserver)
	params.Set("class", "user_defined")
	params.Set("fields", "policy_group,vserver,class,max_throughput,min_throughput,num_workloads,is_shared")

	raw, err := c.GenericGet(ctx, "/private/cli/qos/policy-group", params, 0)
	if err != nil {
		return nil, err
	}
	var result struct {
		Records []cliFixedRecord `json:"records"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	out := make([]json.RawMessage, 0, len(result.Records))
	for _, r := range result.Records {
		maxXput, err := parseXput(string(r.MaxThroughput))
		if err != nil {
			slog.Warn("skipping fixed QoS policy: failed to parse max_throughput",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		minXput, err := parseXput(string(r.MinThroughput))
		if err != nil {
			slog.Warn("skipping fixed QoS policy: failed to parse min_throughput",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}

		maxIOPS, err := xputIOPS(maxXput)
		if err != nil {
			slog.Warn("skipping fixed QoS policy: failed to convert max_throughput IOPS",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		maxMbps, err := xputMbps(maxXput)
		if err != nil {
			slog.Warn("skipping fixed QoS policy: failed to convert max_throughput Mbps",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		minIOPS, err := xputIOPS(minXput)
		if err != nil {
			slog.Warn("skipping fixed QoS policy: failed to convert min_throughput IOPS",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		minMbps, err := xputMbps(minXput)
		if err != nil {
			slog.Warn("skipping fixed QoS policy: failed to convert min_throughput Mbps",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}

		rec := clusterQoSRecord{
			Name:        r.PolicyGroup,
			SVM:         clusterSVMRef{Name: r.Vserver},
			Scope:       "cluster",
			ObjectCount: r.NumWorkloads,
			Fixed: &clusterFixed{
				MaxThroughputIOPS: maxIOPS,
				MaxThroughputMbps: maxMbps,
				MinThroughputIOPS: minIOPS,
				MinThroughputMbps: minMbps,
				CapacityShared:    r.IsShared,
			},
		}
		b, err := json.Marshal(rec)
		if err != nil {
			slog.Warn("skipping fixed QoS policy: failed to marshal record",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

func (c *Client) GetAdminSVMAdaptivePolicies(ctx context.Context, adminVserver string) ([]json.RawMessage, error) {
	params := url.Values{}
	params.Set("vserver", adminVserver)
	params.Set("fields", "policy_group,vserver,expected_iops,peak_iops,absolute_min_iops,expected_iops_allocation,peak_iops_allocation,block_size,num_workloads")

	raw, err := c.GenericGet(ctx, "/private/cli/qos/adaptive-policy-group", params, 0)
	if err != nil {
		return nil, err
	}
	var result struct {
		Records []cliAdaptiveRecord `json:"records"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	out := make([]json.RawMessage, 0, len(result.Records))
	for _, r := range result.Records {
		expXput, err := parseXput(string(r.ExpectedIOPS))
		if err != nil {
			slog.Warn("skipping adaptive QoS policy: failed to parse expected_iops",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		peakXput, err := parseXput(string(r.PeakIOPS))
		if err != nil {
			slog.Warn("skipping adaptive QoS policy: failed to parse peak_iops",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		absMinXput, err := parseXput(string(r.AbsoluteMinIOPS))
		if err != nil {
			slog.Warn("skipping adaptive QoS policy: failed to parse absolute_min_iops",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		expIOPS, err := xputIOPS(expXput)
		if err != nil {
			slog.Warn("skipping adaptive QoS policy: failed to convert expected_iops",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		peakIOPS, err := xputIOPS(peakXput)
		if err != nil {
			slog.Warn("skipping adaptive QoS policy: failed to convert peak_iops",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		absMinIOPS, err := xputIOPS(absMinXput)
		if err != nil {
			slog.Warn("skipping adaptive QoS policy: failed to convert absolute_min_iops",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}

		rec := clusterQoSRecord{
			Name:        r.PolicyGroup,
			SVM:         clusterSVMRef{Name: r.Vserver},
			Scope:       "cluster",
			ObjectCount: r.NumWorkloads,
			Adaptive: &clusterAdaptive{
				ExpectedIOPS:           expIOPS,
				PeakIOPS:               peakIOPS,
				AbsoluteMinIOPS:        absMinIOPS,
				BlockSize:              strings.ToLower(r.BlockSize),
				ExpectedIOPSAllocation: r.ExpectedIOPSAllocation,
				PeakIOPSAllocation:     r.PeakIOPSAllocation,
			},
		}
		b, err := json.Marshal(rec)
		if err != nil {
			slog.Warn("skipping adaptive QoS policy: failed to marshal record",
				slog.String("policy", r.PolicyGroup), slog.Any("error", err))
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

func (c *Client) CreateQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, &statusCode, responseHeaders).
		BodyJSON(qosPolicy).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy, oldQosPolicyName string, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		qPolicy    ontap.GetData
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", oldQosPolicyName)
	params.Set("svm", svmName)

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&qPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if qPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to update qos policy %s on svm %s because it does not exist", oldQosPolicyName, svmName)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/qos/policies/`+qPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Patch().
		ToBytesBuffer(&buf).
		BodyJSON(qosPolicy)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteQoSPolicy(ctx context.Context, qosPolicy ontap.QoSPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		qPolicy    ontap.GetData
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", qosPolicy.Name)
	params.Set("svm", qosPolicy.SVM.Name)

	builder := c.baseRequestBuilder(`/api/storage/qos/policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&qPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if qPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to delete qos policy %s on svm %s because it does not exist", qosPolicy.Name, qosPolicy.SVM.Name)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/qos/policies/`+qPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
