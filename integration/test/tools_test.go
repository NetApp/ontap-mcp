package main

import (
	"bufio"
	"cmp"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	CheckTools = "CHECK_TOOLS"
	ConfigFile = "ontap.yaml"
	Cluster    = "umeng-aff300-05-06"
	ClusterStr = "On the " + Cluster + " cluster, "
)

type envConfig struct {
	llmUserName  string
	llmToken     string
	llmBaseURL   string
	openaiModel  string
	mcpServerURL string
}

type ontapVerifier struct {
	api            string
	validationFunc func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool
}

type Agent struct {
	userName     string
	openaiClient openai.Client
	mcpSession   *mcp.ClientSession
	mcpClient    *mcp.Client
	tools        []*mcp.Tool
	model        string
}

func TestOntapMCPTools(t *testing.T) {
	SkipIfMissing(t, CheckTools)
	envConfigData, err := loadEnv()
	if err != nil {
		t.Errorf("Failed to fetch env vars %v", err)
		return
	}

	agent, err := NewAgent(envConfigData.llmUserName, envConfigData.llmToken, envConfigData.llmBaseURL, envConfigData.openaiModel, envConfigData.mcpServerURL)
	if err != nil {
		slog.Error("Failed to create agent", slog.Any("error", err))
	}
	defer agent.Close()

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		// Cluster operations
		{
			name:             "List all clusters",
			input:            "List all clusters",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},

		// Volume operations
		{
			name:             "List all volumes in one cluster in one svm with given fields",
			input:            ClusterStr + "for every volume on the marketing svm, show me the name, used size, available size, and snapshot policy",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?svm=marketing", validationFunc: listObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume docs in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume docsnew in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docsnew&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "resize the docs volume on the marketing svm to 25MB",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "update junction path of the docs volume on the marketing svm to empty",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Enable volume autogrowth",
			input:            ClusterStr + "enable autogrowth and grow percent to 62 on the docs volume in the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename volume",
			input:            ClusterStr + "rename the docs volume on the marketing svm to docsnew",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docsnew&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the docsnew volume on the marketing svm to offline",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the docsnew volume on the marketing svm to online",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume junction path",
			input:            ClusterStr + "update junction path of the docsnew volume on the marketing svm to /docs",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "List one volume in one cluster in one svm with specific field",
			input:            ClusterStr + "for docsnew volume on the marketing svm, show me the name and junction path",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docsnew&svm=marketing", validationFunc: listObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume docs in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume docsnew in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docsnew&svm=marketing", validationFunc: deleteObject},
		},

		// NFS export policy operations
		{
			name:             "Clean NFS export policy",
			input:            ClusterStr + "delete nfsEngPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsEngPolicy", validationFunc: deleteObject},
		},
		{
			name:             "Clean NFS export policy",
			input:            ClusterStr + "delete nfsMgrPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsMgrPolicy", validationFunc: deleteObject},
		},
		{
			name:             "Create NFS export policy",
			input:            ClusterStr + "create an NFS export policy name nfsEngPolicy on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsEngPolicy", validationFunc: createObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Attach NFS export policy to volume",
			input:            ClusterStr + "apply nfsEngPolicy NFS export policy to the docs volume in the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename NFS export policy",
			input:            ClusterStr + "rename the NFS export policy from nfsEngPolicy to nfsMgrPolicy on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsMgrPolicy", validationFunc: createObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume docs in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean NFS export policy",
			input:            ClusterStr + "delete nfsMgrPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/nfs/export-policies?name=nfsMgrPolicy", validationFunc: deleteObject},
		},

		// QoS policy operations
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete gold QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=gold", validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete silver QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=silver", validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete payroll QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=payroll", validationFunc: deleteObject},
		},
		{
			name:             "Create fixed QoS policy",
			input:            ClusterStr + "create a fixed QoS policy named gold on the marketing svm with a max throughput of 5000 iops",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=gold", validationFunc: createObject},
		},
		{
			name:             "Create adaptive QoS policy",
			input:            ClusterStr + "create a adaptive QoS policy named payroll on the marketing svm with a expected iops as 2000 peak iops as 5000 and absolute min iops is 10",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=payroll", validationFunc: createObject},
		},
		{
			name:             "Rename QoS policy",
			input:            ClusterStr + "rename the QoS policy from gold to silver on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=silver", validationFunc: createObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete silver QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=silver", validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy",
			input:            ClusterStr + "delete payroll QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=payroll", validationFunc: deleteObject},
		},

		// QoS policy assign to volume operations
		{
			name:             "Clean QoS policy qos_docs_200iops",
			input:            ClusterStr + "delete qos_docs_200iops QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=qos_docs_200iops", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos",
			input:            ClusterStr + "delete volume docs_qos in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos2",
			input:            ClusterStr + "delete volume docs_qos2 in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos2&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos3",
			input:            ClusterStr + "delete volume docs_qos3 in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos3&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Create fixed QoS policy qos_docs_200iops",
			input:            ClusterStr + "create a fixed QoS policy named qos_docs_200iops on the marketing svm with a max throughput of 200 iops and min throughput of 0 iops",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=qos_docs_200iops", validationFunc: createObject},
		},
		{
			name:             "Create volume docs_qos",
			input:            ClusterStr + "create a 20MB volume named docs_qos on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos&svm=marketing", validationFunc: createObject},
		},
		{
			name:             "Apply named QoS policy to existing volume",
			input:            ClusterStr + "apply the qos_docs_200iops QoS policy to the docs_qos volume on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSAssigned("qos_docs_200iops")},
		},
		{
			name:             "Create volume docs_qos2 with named QoS policy",
			input:            ClusterStr + "create a 20MB volume named docs_qos2 on the marketing svm and the harvest_vc_aggr aggregate with QoS policy qos_docs_200iops",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos2&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSAssigned("qos_docs_200iops")},
		},
		{
			name:             "Create volume docs_qos3 with inline QoS max 300 iops",
			input:            ClusterStr + "create a 20MB volume named docs_qos3 on the marketing svm and the harvest_vc_aggr aggregate with an inline QoS limit of max_iops 300",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos3&svm.name=marketing&fields=qos.policy.max_throughput_iops", validationFunc: verifyQoSMaxIOPS(300)},
		},
		{
			name:             "Update volume docs_qos3 inline QoS to max 150 iops",
			input:            ClusterStr + "update the docs_qos3 volume on the marketing svm setting an inline QoS limit of max_iops 150",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos3&svm.name=marketing&fields=qos.policy.max_throughput_iops", validationFunc: verifyQoSMaxIOPS(150)},
		},
		// Mode switching: inline → named
		{
			name:             "Switch docs_qos3 from inline to named QoS policy",
			input:            ClusterStr + "apply the qos_docs_200iops QoS policy to the docs_qos3 volume on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos3&svm.name=marketing&fields=qos.policy.name", validationFunc: verifyQoSAssigned("qos_docs_200iops")},
		},
		// Mode switching: named → inline
		{
			name:             "Switch docs_qos from named policy to inline QoS",
			input:            ClusterStr + "update the docs_qos volume on the marketing svm setting an inline QoS limit of max_iops 100",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos&svm.name=marketing&fields=qos.policy.max_throughput_iops", validationFunc: verifyQoSMaxIOPS(100)},
		},
		// Cleanup
		{
			name:             "Clean volume docs_qos after test",
			input:            ClusterStr + "delete volume docs_qos in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos2 after test",
			input:            ClusterStr + "delete volume docs_qos2 in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos2&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean volume docs_qos3 after test",
			input:            ClusterStr + "delete volume docs_qos3 in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=docs_qos3&svm=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Clean QoS policy qos_docs_200iops after test",
			input:            ClusterStr + "delete qos_docs_200iops QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/qos/policies?name=qos_docs_200iops", validationFunc: deleteObject},
		},

		// CIFS share operations
		{
			name:             "Clean CIFS share",
			input:            ClusterStr + "delete cifsFin CIFS share in vs_test4 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/cifs/shares?name=cifsFin", validationFunc: deleteObject},
		},
		{
			name:             "Create CIFS share",
			input:            ClusterStr + "create CIFS share named cifsFin with path as / on the vs_test4 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/cifs/shares?name=cifsFin", validationFunc: createObject},
		},
		{
			name:             "Update CIFS share",
			input:            ClusterStr + "update CIFS share cifsFin path to /vol_test2 on the vs_test4 svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean CIFS share",
			input:            ClusterStr + "delete cifsFin CIFS share in vs_test4 svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/cifs/shares?name=cifsFin", validationFunc: deleteObject},
		},

		// Snapshot policy operations
		{
			name:             "Clean snapshot policy every4hours",
			input:            ClusterStr + "delete every4hours snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=every4hours", validationFunc: deleteObject},
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            ClusterStr + "Delete every5min snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=every5min", validationFunc: deleteObject},
		},
		{
			name:             "Create snapshot policy every4hours",
			input:            ClusterStr + "create a snapshot policy named every4hours on the marketing SVM. The schedule is 4hours and keeps the last 5 snapshots",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=every4hours", validationFunc: createObject},
		},
		{
			name:             "Create snapshot policy every5min",
			input:            ClusterStr + "create a snapshot policy named every5min on the marketing SVM. The schedule is 5minutes and keeps the last 2 snapshots",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=every5min", validationFunc: createObject},
		},
		{
			name:             "Clean snapshot policy every4hours",
			input:            ClusterStr + "delete every4hours snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=every4hours", validationFunc: deleteObject},
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            ClusterStr + "Delete every5min snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=every5min", validationFunc: deleteObject},
		},

		// Iscsi service operations
		{
			name:             "Clean Iscsi service",
			input:            ClusterStr + "delete iscsi service in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=marketing", validationFunc: deleteObject},
		},
		{
			name:             "Create Iscsi service",
			input:            ClusterStr + "create iscsi service target named alias tgpath on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=marketing", validationFunc: createObject},
		},
		{
			name:             "Update Iscsi service",
			input:            ClusterStr + "disabled iscsi service on the marketing svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean Iscsi service",
			input:            ClusterStr + "delete iscsi service in marketing svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/protocols/san/iscsi/services?svm.name=marketing", validationFunc: deleteObject},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Errorf("Error parsing the config: %v\n", err)
		return
	}

	poller := cfg.Pollers[Cluster]
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: poller.UseInsecureTLS, // #nosec G402
		},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			if err = agent.Chat(context.Background(), t, tt.input, tt.expectedOntapErr); err != nil {
				slog.Error("Error processing input", slog.Any("error", err))
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %s", slog.Any("input", tt.input))
			}
		})
	}
}

func NewAgent(llmUserName, llmToken, llmBaseURL, openaiModel, mcpServerURL string) (*Agent, error) {
	openaiClient := openai.NewClient(
		option.WithAPIKey(llmToken),
		option.WithBaseURL(llmBaseURL),
	)

	impl := &mcp.Implementation{
		Name:    "ontap-mcp-agent",
		Version: "1.0.0",
	}
	mcpClient := mcp.NewClient(impl, nil)

	mcpTransport := &mcp.StreamableClientTransport{
		Endpoint:   mcpServerURL,
		HTTPClient: &http.Client{},
	}

	ctx := context.Background()
	session, err := mcpClient.Connect(ctx, mcpTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list all tools: %w", err)
	}

	slog.Info("Connected to MCP server", slog.Int("Found tools", len(toolsResult.Tools)))

	return &Agent{
		userName:     llmUserName,
		openaiClient: openaiClient,
		mcpSession:   session,
		mcpClient:    mcpClient,
		tools:        toolsResult.Tools,
		model:        openaiModel,
	}, nil
}

func (a *Agent) convertMCPToolsToOpenAI() []openai.ChatCompletionToolUnionParam {
	tools := make([]openai.ChatCompletionToolUnionParam, len(a.tools))
	for i, tool := range a.tools {
		tools[i] = openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
			Parameters:  tool.InputSchema.(map[string]any),
		})
	}
	return tools
}

func (a *Agent) callMCPTool(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	result, err := a.mcpSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool: %w", err)
	}

	if result.IsError {
		var errorMsg strings.Builder
		for _, content := range result.Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				errorMsg.WriteString(textContent.Text)
			}
		}
		return "", fmt.Errorf("error running the tool: %s", errorMsg.String())
	}

	var output strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			output.WriteString(textContent.Text)
		}
	}

	return output.String(), nil
}

func (a *Agent) Chat(ctx context.Context, t *testing.T, userMessage string, expectedOntapErrorStr string) error {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(userMessage),
	}

	tools := a.convertMCPToolsToOpenAI()
	maxIterations := 10

	for range maxIterations {
		completion, err := a.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    a.model,
			Messages: messages,
			Tools:    tools,
			User:     openai.String(a.userName), // Required by NetApp proxy
		})

		if err != nil {
			return fmt.Errorf("OpenAI error: %w", err)
		}

		assistantMessage := completion.Choices[0].Message

		if len(assistantMessage.ToolCalls) == 0 {
			return nil
		}

		messages = append(messages, assistantMessage.ToParam())

		for _, toolCall := range assistantMessage.ToolCalls {
			toolName := toolCall.Function.Name

			var args map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return fmt.Errorf("failed to parse tool args: %w", err)
			}

			slog.Debug("", slog.String("Calling tool", toolName), slog.Any("args", args))

			result, err := a.callMCPTool(ctx, toolName, args)
			// Only error out when we don't expect the error scenarios
			if err != nil {
				if expectedOntapErrorStr == "" || !strings.Contains(err.Error(), expectedOntapErrorStr) {
					t.Errorf("Error: %s %s %v", slog.String("tool", toolName), slog.Any("args", args), slog.Any("err", err))
				}
			}

			slog.Debug("", slog.Any("Tool result", result))

			messages = append(messages, openai.ToolMessage(result, toolCall.ID))
		}
	}

	return nil
}

func (a *Agent) Close() {
	if a.mcpSession != nil {
		//goland:noinspection GoUnhandledErrorResult
		err := a.mcpSession.Close()
		if err != nil {
			return
		}
	}
}

func loadEnv() (envConfig, error) {
	file, err := os.Open(".ontap-mcp.env")
	if err != nil {
		// .ontap-mcp.env file doesn't exist, that's okay
		slog.Error(".ontap-mcp.env file not exist", slog.Any("error", err))
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("failed to close file", slog.Any("error", err))
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				if err = os.Setenv(key, value); err != nil {
					// Log the error and proceed further
					slog.Error("Error setting environment variable", slog.String("key", key), slog.String("value", value), slog.Any("err", err))
				}
			}
		}
	}

	llmUserName := os.Getenv("LLM_USER")
	if llmUserName == "" {
		slog.Error("LLM_USER environment variable is required. Set it in .ontap-mcp.env file or export it.")
		return envConfig{}, errors.New("LLM_USER environment variable is required")
	}

	llmToken := os.Getenv("LLM_TOKEN")
	if llmToken == "" {
		slog.Error("LLM_TOKEN environment variable is required. Get it from https://llm-proxy-api.ai.openeng.netapp.com/ui/")
		return envConfig{}, errors.New("LLM_TOKEN environment variable is required")
	}

	llmBaseURL := cmp.Or(os.Getenv("LLM_PROXY"), "https://llm-proxy-api.ai.openeng.netapp.com/v1")
	slog.Debug("", slog.String("LLM PROXY Base URL", llmBaseURL))

	openaiModel := cmp.Or(os.Getenv("OPENAI_MODEL"), "gpt-5.2-chat")
	slog.Debug("", slog.String("Model", openaiModel))

	mcpServerURL := cmp.Or(os.Getenv("MCP_URL"), "http://localhost:8083")
	slog.Debug("", slog.String("Mcp server url", mcpServerURL))

	return envConfig{llmUserName: llmUserName, llmToken: llmToken, llmBaseURL: llmBaseURL, openaiModel: openaiModel, mcpServerURL: mcpServerURL}, nil
}

func SkipIfMissing(t *testing.T, vars ...string) {
	t.Helper()
	anyMatches := false
	for _, v := range vars {
		e := os.Getenv(v)
		if e != "" {
			anyMatches = true
			break
		}
	}
	if !anyMatches {
		t.Skipf("Set one of %s envvars to run the test", strings.Join(vars, ", "))
	}
}

func createObject(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	var data ontap.GetData
	err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		ToJSON(&data).
		Fetch(context.Background())
	if err != nil {
		t.Errorf("createObject: request failed: %v", err)
		return false
	}
	if data.NumRecords != 1 {
		t.Errorf("No records found")
		return false
	}
	return true
}

func deleteObject(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	var data ontap.GetData
	err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		ToJSON(&data).
		Fetch(context.Background())
	if err != nil {
		t.Errorf("deleteObject: request failed: %v", err)
		return false
	}
	if data.NumRecords != 0 {
		t.Errorf("Records should not be found")
		return false
	}
	return true
}

func verifyQoSAssigned(policyName string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type qosPolicy struct {
			Name string `json:"name"`
		}
		type qos struct {
			Policy qosPolicy `json:"policy"`
		}
		type volumeRecord struct {
			QoS qos `json:"qos"`
		}
		type response struct {
			NumRecords int            `json:"num_records"`
			Records    []volumeRecord `json:"records"`
		}

		var data response
		err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyQoSAssigned: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyQoSAssigned: expected 1 record, got %d", data.NumRecords)
			return false
		}
		got := data.Records[0].QoS.Policy.Name
		if got != policyName {
			t.Errorf("verifyQoSAssigned: qos.policy.name = %q, want %q", got, policyName)
			return false
		}
		return true
	}
}

func verifyQoSMaxIOPS(wantIOPS int) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type qosPolicy struct {
			MaxThroughIOPS int `json:"max_throughput_iops"`
		}
		type qos struct {
			Policy qosPolicy `json:"policy"`
		}
		type volumeRecord struct {
			QoS qos `json:"qos"`
		}
		type response struct {
			NumRecords int            `json:"num_records"`
			Records    []volumeRecord `json:"records"`
		}

		var data response
		err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyQoSMaxIOPS: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyQoSMaxIOPS: expected 1 record, got %d", data.NumRecords)
			return false
		}
		got := data.Records[0].QoS.Policy.MaxThroughIOPS
		if got != wantIOPS {
			t.Errorf("verifyQoSMaxIOPS: qos.policy.max_throughput_iops = %d, want %d", got, wantIOPS)
			return false
		}
		return true
	}
}

func listObject(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	err := requests.URL(fmt.Sprintf("https://%s/%s", poller.Addr, api)).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		Fetch(context.Background())
	if err != nil {
		t.Errorf("listObject: request failed: %v", err)
		return false
	}
	return true
}
