package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
)

const (
	CheckTools  = "CHECK_TOOLS"
	CLUSTER_STR = "On the umeng-aff300-05-06 cluster, "
)

type Agent struct {
	userName     string
	openaiClient openai.Client // Not a pointer!
	mcpSession   *mcp.ClientSession
	mcpClient    *mcp.Client
	tools        []*mcp.Tool
	model        string
}

func TestOntapMCPTools(t *testing.T) {
	SkipIfMissing(t, CheckTools)
	llmUserName, llmToken, llmBaseURL, openaiModel, mcpServerUrl := loadEnv()

	agent, err := NewAgent(llmUserName, llmToken, llmBaseURL, openaiModel, mcpServerUrl)
	if err != nil {
		slog.Error("Failed to create agent: %v", slog.Any("error", err))
	}
	defer agent.Close()

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
	}{
		// Cluster operations
		{
			name:             "List all clusters",
			input:            "List all clusters",
			expectedOntapErr: "",
		},

		// Volume operations
		{
			name:             "List all volumes in one cluster",
			input:            CLUSTER_STR + "List all volumes",
			expectedOntapErr: "",
		},
		{
			name:             "List all volumes in wrong cluster",
			input:            "List all volumes in valso cluster",
			expectedOntapErr: "cluster valso not found",
		},
		{
			name:             "List all volumes in one cluster in one svm",
			input:            CLUSTER_STR + "List all volumes in vs_test svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean volume",
			input:            CLUSTER_STR + "delete volume docs in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean volume",
			input:            CLUSTER_STR + "delete volume docsnew in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Create volume",
			input:            CLUSTER_STR + "create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
		},
		{
			name:             "Update volume size",
			input:            CLUSTER_STR + "resize the docs volume on the marketing svm to 25MB",
			expectedOntapErr: "",
		},
		{
			name:             "Enable volume autogrowth",
			input:            CLUSTER_STR + "enable autogrowth and grow percent to 62 on the docs volume in the marketing svm",
			expectedOntapErr: "",
		},
		{
			name:             "Rename volume",
			input:            CLUSTER_STR + "rename the docs volume on the marketing svm to docsnew",
			expectedOntapErr: "",
		},
		{
			name:             "Update volume state",
			input:            CLUSTER_STR + "update state of the docsnew volume on the marketing svm to offline",
			expectedOntapErr: "",
		},
		{
			name:             "Clean volume",
			input:            CLUSTER_STR + "delete volume docsnew in marketing svm",
			expectedOntapErr: "because it does not exist",
		},

		// NFS export policy operations
		{
			name:             "List all NFS export policies",
			input:            CLUSTER_STR + "List all NFS export policies",
			expectedOntapErr: "",
		},
		{
			name:             "Clean NFS export policy",
			input:            CLUSTER_STR + "delete nfsEngPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean NFS export policy",
			input:            CLUSTER_STR + "delete nfsMgrPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Create NFS export policy",
			input:            CLUSTER_STR + "create an NFS export policy name nfsEngPolicy on the marketing svm",
			expectedOntapErr: "",
		},
		{
			name:             "Create volume",
			input:            CLUSTER_STR + "create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate",
			expectedOntapErr: "",
		},
		{
			name:             "Attach NFS export policy to volume",
			input:            CLUSTER_STR + "apply nfsEngPolicy NFS export policy to the docs volume in the marketing svm",
			expectedOntapErr: "",
		},
		{
			name:             "Rename NFS export policy",
			input:            CLUSTER_STR + "rename the NFS export policy from nfsEngPolicy to nfsMgrPolicy on the marketing svm",
			expectedOntapErr: "",
		},
		{
			name:             "Clean volume",
			input:            CLUSTER_STR + "delete volume docs in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean NFS export policy",
			input:            CLUSTER_STR + "delete nfsMgrPolicy NFS export policy",
			expectedOntapErr: "because it does not exist",
		},

		// QoS policy operations
		{
			name:             "List QoS policies",
			input:            CLUSTER_STR + "List all QoS policies",
			expectedOntapErr: "",
		},
		{
			name:             "Clean QoS policy",
			input:            CLUSTER_STR + "delete gold QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean QoS policy",
			input:            CLUSTER_STR + "delete silver QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean QoS policy",
			input:            CLUSTER_STR + "delete payroll QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Create fixed QoS policy",
			input:            CLUSTER_STR + "create a fixed QoS policy named gold on the marketing svm with a max throughput of 5000 iops",
			expectedOntapErr: "",
		},
		{
			name:             "Create adaptive QoS policy",
			input:            CLUSTER_STR + "create a adaptive QoS policy named payroll on the marketing svm with a expected iops as 2000 peak iops as 5000 and absolute min iops is 10",
			expectedOntapErr: "",
		},
		{
			name:             "Rename QoS policy",
			input:            CLUSTER_STR + "rename the QoS policy from gold to silver on the marketing svm",
			expectedOntapErr: "",
		},
		{
			name:             "Clean QoS policy",
			input:            CLUSTER_STR + "delete silver QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean QoS policy",
			input:            CLUSTER_STR + "delete payroll QoS policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},

		// CIFS share operations
		{
			name:             "List CIFS share",
			input:            CLUSTER_STR + "List all CIFS shares",
			expectedOntapErr: "",
		},
		{
			name:             "Clean CIFS share",
			input:            CLUSTER_STR + "delete cifsFin CIFS share in vs_test4 svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Create CIFS share",
			input:            CLUSTER_STR + "create CIFS share named cifsFin with path as / on the vs_test4 svm",
			expectedOntapErr: "",
		},
		{
			name:             "Update CIFS share",
			input:            CLUSTER_STR + "update CIFS share cifsFin path to /vol_test2 on the vs_test4 svm",
			expectedOntapErr: "",
		},
		{
			name:             "Clean CIFS share",
			input:            CLUSTER_STR + "delete cifsFin CIFS share in vs_test4 svm",
			expectedOntapErr: "because it does not exist",
		},

		// Snapshot policy operations
		{
			name:             "List snapshot policies",
			input:            CLUSTER_STR + "List all snapshot policies",
			expectedOntapErr: "",
		},
		{
			name:             "Clean snapshot policy every4hours",
			input:            CLUSTER_STR + "delete every4hours snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            CLUSTER_STR + "Delete every5min snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Create snapshot policy every4hours",
			input:            CLUSTER_STR + "create a snapshot policy named every4hours on the marketing SVM. The schedule is 4hours and keeps the last 5 snapshots",
			expectedOntapErr: "",
		},
		{
			name:             "Create snapshot policy every5min",
			input:            CLUSTER_STR + "create a snapshot policy named every5min on the marketing SVM. The schedule is 5minutes and keeps the last 2 snapshots",
			expectedOntapErr: "",
		},
		{
			name:             "Clean snapshot policy every4hours",
			input:            CLUSTER_STR + "delete every4hours snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            CLUSTER_STR + "Delete every5min snapshot policy in marketing svm",
			expectedOntapErr: "because it does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			if _, err = agent.Chat(context.Background(), tt.input, tt.expectedOntapErr, t); err != nil {
				slog.Error("Error processing input: %v", slog.Any("error", err))
			}
		})
	}
}

func NewAgent(llmUserName, llmToken, llmBaseURL, openaiModel, mcpServerUrl string) (*Agent, error) {
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
		Endpoint:   mcpServerUrl,
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
		parameters := openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]interface{}{},
		}

		if tool.InputSchema != nil {
			if schema, ok := tool.InputSchema.(map[string]interface{}); ok {
				if schema["type"] == "object" {
					if _, hasProps := schema["properties"]; !hasProps {
						schema["properties"] = map[string]interface{}{}
					}
				}
				parameters = openai.FunctionParameters(schema)
			}
		}

		tools[i] = openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
			Parameters:  parameters,
		})
	}
	return tools
}

func (a *Agent) callMCPTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	result, err := a.mcpSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool: %w", err)
	}

	if result.IsError {
		var errorMsg string
		for _, content := range result.Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				errorMsg += textContent.Text
			}
		}
		return "", fmt.Errorf("error running the tool: %s", errorMsg)
	}

	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			output += textContent.Text
		}
	}

	return output, nil
}

func (a *Agent) Chat(ctx context.Context, userMessage string, expectedOntapErrorStr string, t *testing.T) (string, error) {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(userMessage),
	}

	tools := a.convertMCPToolsToOpenAI()
	maxIterations := 10

	for iteration := 0; iteration < maxIterations; iteration++ {
		completion, err := a.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    a.model,
			Messages: messages,
			Tools:    tools,
			User:     openai.String(a.userName), // Required by NetApp proxy
		})

		if err != nil {
			return "", fmt.Errorf("OpenAI error: %w", err)
		}

		assistantMessage := completion.Choices[0].Message

		if len(assistantMessage.ToolCalls) == 0 {
			return assistantMessage.Content, nil
		}

		messages = append(messages, assistantMessage.ToParam())

		for _, toolCall := range assistantMessage.ToolCalls {
			toolName := toolCall.Function.Name

			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("failed to parse tool args: %w", err)
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

	return "Maximum iterations reached", nil
}

func (a *Agent) Close() {
	if a.mcpSession != nil {
		a.mcpSession.Close()
	}
}

func loadEnv() (string, string, string, string, string) {
	file, err := os.Open(".ontap-mcp.env")
	if err != nil {
		// .ontap-mcp.env file doesn't exist, that's okay
		slog.Debug(".ontap-mcp.env file not exist")
	}
	defer file.Close()

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
				os.Setenv(key, value)
			}
		}
	}

	llmUserName := os.Getenv("LLM_USER")
	if llmUserName == "" {
		slog.Error("LLM_USER environment variable is required. Set it in .ontap-mcp.env file or export it.")
	}

	llmToken := os.Getenv("LLM_TOKEN")
	if llmToken == "" {
		slog.Error("LLM_TOKEN environment variable is required. Get it from https://llm-proxy-api.ai.openeng.netapp.com/ui/")
	}

	llmBaseURL := os.Getenv("LLM_PROXY")
	if llmBaseURL == "" {
		llmBaseURL = "https://llm-proxy-api.ai.openeng.netapp.com/v1"
	}
	slog.Debug("", slog.String("LLM PROXY Base URL", llmBaseURL))

	openaiModel := os.Getenv("OPENAI_MODEL")
	if openaiModel == "" {
		openaiModel = "gpt-5.2-chat"
	}
	slog.Debug("", slog.String("Model", openaiModel))

	mcpServerUrl := os.Getenv("MCP_URL")
	if mcpServerUrl == "" {
		mcpServerUrl = "http://localhost:8083"
	}
	slog.Debug("", slog.String("Mcp server url", mcpServerUrl))
	return llmUserName, llmToken, llmBaseURL, openaiModel, mcpServerUrl
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
