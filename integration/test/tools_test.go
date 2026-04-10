package main

import (
	"bufio"
	"cmp"
	"context"
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
	CheckTools         = "CHECK_TOOLS"
	OpenaiModelVersion = "gpt-4.1"
	ConfigFile         = "ontap.yaml"
	Cluster            = "umeng-aff300-05-06"
	ClusterStr         = "On the " + Cluster + " cluster, "
)

var testAgent *Agent
var testPrefix string

func rn(base string) string {
	if testPrefix == "" {
		return base
	}
	return base + "_" + testPrefix
}

func TestMain(m *testing.M) {
	testPrefix = os.Getenv("TEST_SUFFIX")
	slog.Info("Integration test run", slog.String("TEST_SUFFIX", testPrefix))
	if os.Getenv(CheckTools) != "" {
		envConfigData, err := loadEnv()
		if err != nil {
			slog.Error("Failed to load env", slog.Any("error", err))
			os.Exit(1)
		}
		testAgent, err = NewAgent(envConfigData.llmUserName, envConfigData.llmToken, envConfigData.llmBaseURL, envConfigData.openaiModel, envConfigData.mcpServerURL)
		if err != nil {
			slog.Error("Failed to create agent", slog.Any("error", err))
			os.Exit(1)
		}
	}
	code := m.Run()
	if testAgent != nil {
		testAgent.Close()
	}
	os.Exit(code)
}

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
		HTTPClient: &http.Client{Timeout: 2 * time.Minute},
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

func (a *Agent) ChatWithResponse(ctx context.Context, t *testing.T, userMessage string, expectedOntapErrorStr string) (string, error) {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(userMessage),
	}

	tools := a.convertMCPToolsToOpenAI()
	maxIterations := 10

	var errFound error
	failedTool := ""
	argsUsed := make(map[string]any)

	for range maxIterations {
		completion, err := a.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    a.model,
			Messages: messages,
			Tools:    tools,
			User:     openai.String(a.userName),
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

			var args map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("failed to parse tool args: %w", err)
			}

			slog.Debug("", slog.String("Calling tool", toolName), slog.Any("args", args))

			result, err := a.callMCPTool(ctx, toolName, args)
			if err != nil {
				if expectedOntapErrorStr != "" && strings.Contains(err.Error(), expectedOntapErrorStr) {
					return "", nil // test passed, expected error was observed
				}
				failedTool = toolName
				argsUsed = args
				errFound = err
				slog.Warn("LLM will retry", slog.String("tool", toolName), slog.Any("args", args), slog.Any("error", err))

				result = "Error: " + err.Error()
			}

			slog.Debug("", slog.Any("Tool result", result))
			messages = append(messages, openai.ToolMessage(result, toolCall.ID))
		}
	}

	t.Errorf("Tool %q args %v returned error %v", failedTool, argsUsed, errFound)
	return "", fmt.Errorf("max iterations (%d) reached; last tool %q args %v error: %w", maxIterations, failedTool, argsUsed, errFound)
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

	openaiModel := OpenaiModelVersion
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
	err := requests.URL("https://"+poller.Addr+"/"+api).
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
	err := requests.URL("https://"+poller.Addr+"/"+api).
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

func listObject(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	err := requests.URL("https://"+poller.Addr+"/"+api).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		Fetch(context.Background())
	if err != nil {
		t.Errorf("listObject: request failed: %v", err)
		return false
	}
	return true
}
