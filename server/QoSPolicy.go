package server

import (
	"context"
	"errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
	"strings"
)

func (a *App) ListQoSPolicies(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicy) (*mcp.CallToolResult, any, error) {
	qosPolicyGet := newGetQoSPolicy(parameters)
	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	qosPolicies, err := client.GetQoSPolicy(ctx, qosPolicyGet)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(qosPolicies, ",")},
		},
	}, nil, nil
}

func (a *App) CreateQoSPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicy) (*mcp.CallToolResult, any, error) {
	qosPolicyCreate, err := newCreateQoSPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateQoSPolicy(ctx, qosPolicyCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "QoS Policy created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateQosPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicy) (*mcp.CallToolResult, any, error) {
	qosPolicyUpdate, err := newUpdateQoSPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateQoSPolicy(ctx, qosPolicyUpdate, parameters.Name, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "QoS Policy updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) DeleteQoSPolicy(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.QoSPolicy) (*mcp.CallToolResult, any, error) {
	qosPolicyDelete, err := newDeleteQoSPolicy(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteQoSPolicy(ctx, qosPolicyDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "QoS policy deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// newGetQoSPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newGetQoSPolicy(in tool.QoSPolicy) ontap.QoSPolicy {
	out := ontap.QoSPolicy{}
	if in.SVM != "" {
		out.SVM = ontap.NameAndUUID{Name: in.SVM}
	}

	return out
}

// newCreateQoSPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateQoSPolicy(in tool.QoSPolicy) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("qos policy name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	if in.MaxThIOPS != "" || in.MinThIOPS != "" {
		if in.MaxThIOPS == "" {
			return out, errors.New("max throughput iops is required")
		}
		if in.MinThIOPS == "" {
			return out, errors.New("min throughput iops is required")
		}

		maxiops, err := parseSize(in.MaxThIOPS)
		if err != nil {
			return out, err
		}
		miniops, err := parseSize(in.MinThIOPS)
		if err != nil {
			return out, err
		}
		out.Fixed = ontap.QoSFixed{
			MaxThIOPS: maxiops,
			MinThIOPS: miniops,
		}
	} else {
		if in.ExpectedIOPS == "" {
			return out, errors.New("expected iops is required")
		}
		if in.PeakIOPS == "" {
			return out, errors.New("peak iops is required")
		}
		if in.AbsoluteMinIOPS == "" {
			return out, errors.New("absolute min iops is required")
		}

		expectediops, err := parseSize(in.ExpectedIOPS)
		if err != nil {
			return out, err
		}
		peakiops, err := parseSize(in.PeakIOPS)
		if err != nil {
			return out, err
		}
		absoluteMiniops, err := parseSize(in.AbsoluteMinIOPS)
		if err != nil {
			return out, err
		}
		out.Adaptive = ontap.QoSAdaptive{
			ExpectedIOPS:    expectediops,
			PeakIOPS:        peakiops,
			AbsoluteMinIOPS: absoluteMiniops,
		}
	}

	return out, nil
}

// newUpdateQoSPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateQoSPolicy(in tool.QoSPolicy) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" && in.NewName == "" {
		return out, errors.New("qos policy name is required")
	}

	if in.NewName != "" {
		out.Name = in.NewName
	}

	if in.MaxThIOPS != "" || in.MinThIOPS != "" {
		if in.MaxThIOPS == "" {
			return out, errors.New("max throughput iops is required")
		}
		if in.MinThIOPS == "" {
			return out, errors.New("min throughput iops is required")
		}

		maxiops, err := parseSize(in.MaxThIOPS)
		if err != nil {
			return out, err
		}
		miniops, err := parseSize(in.MinThIOPS)
		if err != nil {
			return out, err
		}
		out.Fixed = ontap.QoSFixed{
			MaxThIOPS: maxiops,
			MinThIOPS: miniops,
		}
	} else if in.ExpectedIOPS != "" || in.PeakIOPS != "" || in.AbsoluteMinIOPS != "" {
		if in.ExpectedIOPS == "" {
			return out, errors.New("expected iops is required")
		}
		if in.PeakIOPS == "" {
			return out, errors.New("peak iops is required")
		}
		if in.AbsoluteMinIOPS == "" {
			return out, errors.New("absolute min iops is required")
		}

		expectediops, err := parseSize(in.ExpectedIOPS)
		if err != nil {
			return out, err
		}
		peakiops, err := parseSize(in.PeakIOPS)
		if err != nil {
			return out, err
		}
		absoluteMiniops, err := parseSize(in.AbsoluteMinIOPS)
		if err != nil {
			return out, err
		}
		out.Adaptive = ontap.QoSAdaptive{
			ExpectedIOPS:    expectediops,
			PeakIOPS:        peakiops,
			AbsoluteMinIOPS: absoluteMiniops,
		}
	}

	return out, nil
}

// newDeleteQoSPolicy validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteQoSPolicy(in tool.QoSPolicy) (ontap.QoSPolicy, error) {
	out := ontap.QoSPolicy{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Name == "" {
		return out, errors.New("qos policy name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Name
	return out, nil
}
