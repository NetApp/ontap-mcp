package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const (
	CheckTools = "CHECK_TOOLS"
)

var deleteObjMap = PromptMap{}
var inputPromptMap = PromptMap{}

type PromptMap struct {
	promts       []string
	promptErrMap map[string]ExpectedErr
}
type ExpectedErr struct {
	expectedLLMErr   string
	expectedOntapErr string
}

type Agent struct {
	userName     string
	openaiClient openai.Client // Not a pointer!
	mcpSession   *mcp.ClientSession
	mcpClient    *mcp.Client
	tools        []*mcp.Tool
	model        string
}

func TestOntapMcpTools(t *testing.T) {
	SkipIfMissing(t, CheckTools)
	validateTools(t)
}

func loadInput() {
	// deleteObjMap map of delete prompts and expected LLM error, expected Ontap error if any. This would delete the objects which are being tested in CI.
	// It's deleted before and after the testing to avoid any duplication of objects
	deleteObjMap.promptErrMap = make(map[string]ExpectedErr)
	deleteObjMap.Set("Delete volume docs in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete volume docsnew in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete nfsEngPolicy NFS export policy in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete nfsMgrPolicy NFS export policy in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete gold QoS policy in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete silver QoS policy in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete payroll QoS policy in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	//deleteObjMap.Set("Delete NFS export policy rule where current ro rule is any in nfsMgrPolicy NFS export policy in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete cifsFin CIFS share in vs_test4 svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete every4hours snapshot policy in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete biweekly snapshot policy in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})
	deleteObjMap.Set("Delete every5min snapshot policy in marketing svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "because it does not exist", expectedLLMErr: ""})

	// inputPromptMap map of customer input prompts and expected LLM error, expected Ontap error if any
	inputPromptMap.promptErrMap = make(map[string]ExpectedErr)
	inputPromptMap.Set("List all clusters", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("List all volumes in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("List all volumes in valso cluster", ExpectedErr{expectedOntapErr: "cluster valso not found", expectedLLMErr: ""})
	inputPromptMap.Set("List all volumes in vs_test svm in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})

	// Volume operations
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, resize the docs volume on the marketing svm to 25MB", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, enable autogrowth and grow percent to 62 on the docs volume in the marketing svm", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, rename the docs volume on the marketing svm to docsnew", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, update state of the docsnew volume on the marketing svm to offline", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, update state of the docsnew volume on the marketing svm to online", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})

	// NFS export policy operations
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create an NFS export policy name nfsEngPolicy on the marketing svm", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("List all NFS export policies in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, apply nfsEngPolicy NFS export policy to the docsnew volume in the marketing svm", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, rename the NFS export policy from nfsEngPolicy to nfsMgrPolicy on the marketing svm.", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})

	// NFS export policy rules operations
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create NFS export policy rule as client match 0.0.0.0/0, ro rule any, rw rule any in nfsMgrPolicy on the marketing svm", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, update NFS export policy rule for nfsMgrPolicy export policy ro rule from any to never", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})

	// QoS policy operations
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create a fixed QoS policy named gold on the marketing svm with a max throughput of 5000 iops", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("List all QoS policies in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create a adaptive QoS policy named payroll on the marketing svm with a expected iops as 2000 peak iops as 5000 and absolute min iops is 10", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, rename the QoS policy from gold to silver on the marketing svm", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})

	// CIFS share operations
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create CIFS share named cifsFin with path as / on the vs_test4 svm", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("List all CIFS shares in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, update CIFS share cifsFin path to /vol_test2 on the vs_test4 svm", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})

	// Snapshot policy operations
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create a snapshot policy named every4hours on the marketing SVM. The schedule is 4hours and keeps the last 5 snapshots", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("List all snapshot policies in umeng-aff300-05-06 cluster", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
	inputPromptMap.Set("On the umeng-aff300-05-06 cluster, create a snapshot policy named every5min on the marketing SVM. The schedule is 5minutes and keeps the last 2 snapshots", ExpectedErr{expectedOntapErr: "", expectedLLMErr: ""})
}

func (p *PromptMap) Set(input string, exptErr ExpectedErr) {
	p.promptErrMap[input] = exptErr
	p.promts = append(p.promts, input)
}

func NewAgent(userName, token string) (*Agent, error) {
	baseURL := os.Getenv("LLM_PROXY")
	if baseURL == "" {
		baseURL = "https://llm-proxy-api.ai.openeng.netapp.com/v1"
	}
	slog.Debug("", slog.String("LLM PROXY Base URL", baseURL))

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-5.2-chat"
	}
	slog.Debug("", slog.String("Model", model))

	mcpServerUrl := os.Getenv("MCP_URL")
	if mcpServerUrl == "" {
		mcpServerUrl = "http://localhost:8083"
	}
	slog.Debug("", slog.String("Mcp server url", mcpServerUrl))

	openaiClient := openai.NewClient(
		option.WithAPIKey(token),
		option.WithBaseURL(baseURL),
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
		userName:     userName,
		openaiClient: openaiClient,
		mcpSession:   session,
		mcpClient:    mcpClient,
		tools:        toolsResult.Tools,
		model:        model,
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

func (a *Agent) Chat(ctx context.Context, userMessage string, expectedLLMErrorStr string, expectedOntapErrorStr string, t *testing.T) (string, error) {
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
			// Only error out when we don't expect the error scenarios
			if expectedLLMErrorStr != "" && !strings.Contains(completion.Choices[0].Message.Content, expectedLLMErrorStr) {
				t.Errorf("Error: %s", slog.Any("err", completion.Choices[0].Message.Content))
			}
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

func loadEnv() {
	file, err := os.Open(".ontap-mcp.env")
	if err != nil {
		// .ontap-mcp.env file doesn't exist, that's okay
		return
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
}

func validateTools(t *testing.T) {
	loadEnv()
	loadInput()

	userName := os.Getenv("LLM_USER")
	if userName == "" {
		slog.Error("LLM_USER environment variable is required. Set it in .ontap-mcp.env file or export it.")
	}

	token := os.Getenv("LLM_TOKEN")
	if token == "" {
		slog.Error("LLM_TOKEN environment variable is required. Get it from https://llm-proxy-api.ai.openeng.netapp.com/ui/")
	}

	agent, err := NewAgent(userName, token)
	if err != nil {
		slog.Error("Failed to create agent: %v", slog.Any("error", err))
	}
	defer agent.Close()

	// Clean the existing objects in the cluster which we would be creating next
	deleteObjects(agent, t)

	// End to end validation of all tool operations
	for _, input := range inputPromptMap.promts {
		expectedErr := inputPromptMap.promptErrMap[input]
		slog.Debug("", slog.String("Input", input))

		ctx := context.Background()
		_, err := agent.Chat(ctx, input, expectedErr.expectedLLMErr, expectedErr.expectedOntapErr, t)
		if err != nil {
			slog.Error("Error processing input: %v", slog.Any("error", err))
		}
	}

	// Safe side, Clean the existing objects in the cluster which we would be creating next
	deleteObjects(agent, t)
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

func deleteObjects(agent *Agent, t *testing.T) {
	for _, deleteInput := range deleteObjMap.promts {
		expectedErr := deleteObjMap.promptErrMap[deleteInput]
		slog.Debug("", slog.String("Input", deleteInput))

		ctx := context.Background()
		_, err := agent.Chat(ctx, deleteInput, expectedErr.expectedLLMErr, expectedErr.expectedOntapErr, t)
		if err != nil {
			t.Errorf("Error detected: %v", slog.Any("err", slog.Any("error", err)))
		}
	}

	slog.Info("Objects were successfully deleted")
}
