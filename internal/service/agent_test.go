package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"go-llm/internal/conversation"
	"go-llm/internal/core"
	"go-llm/internal/events"
	"go-llm/internal/llm"
)

type testToolExecutor struct {
	mu       sync.Mutex
	handlers map[string]func(context.Context) (*llm.ToolResult, error)
}

func newTestToolExecutor() *testToolExecutor {
	return &testToolExecutor{
		handlers: make(map[string]func(context.Context) (*llm.ToolResult, error)),
	}
}

func (e *testToolExecutor) Schemas() []llm.ToolDefinition {
	return nil
}

func (e *testToolExecutor) Execute(
	ctx context.Context,
	name string,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	e.mu.Lock()
	handler, ok := e.handlers[name]
	e.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("unknown test tool %q", name)
	}

	return handler(ctx)
}

func TestChatService_ExecuteTools_PreservesCallOrder(t *testing.T) {

	executor := newTestToolExecutor()

	executor.handlers["tool-a"] = func(ctx context.Context) (*llm.ToolResult, error) {
		time.Sleep(100 * time.Millisecond)

		return &llm.ToolResult{
			Content: "result-a",
		}, nil
	}

	executor.handlers["tool-b"] = func(ctx context.Context) (*llm.ToolResult, error) {
		time.Sleep(50 * time.Millisecond)

		return &llm.ToolResult{
			Content: "result-b",
		}, nil
	}

	executor.handlers["tool-c"] = func(ctx context.Context) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "result-c",
		}, nil
	}

	service := &ChatService{
		executor: executor,
	}

	calls := []llm.ToolCall{
		{
			Function: llm.ToolFunctionCall{
				Name: "tool-a",
			},
		},
		{
			Function: llm.ToolFunctionCall{
				Name: "tool-b",
			},
		},
		{
			Function: llm.ToolFunctionCall{
				Name: "tool-c",
			},
		},
	}

	results, err := service.executeTools(
		context.Background(),
		calls,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 3 {
		t.Fatalf(
			"expected 3 results, got %d",
			len(results),
		)
	}

	expected := []string{
		"tool-a",
		"tool-b",
		"tool-c",
	}

	for i, result := range results {

		if result.Call.Function.Name != expected[i] {
			t.Fatalf(
				"result %d: expected %q, got %q",
				i,
				expected[i],
				result.Call.Function.Name,
			)
		}
	}
}

func TestChatService_ExecuteTools_ReturnsIndividualToolErrors(
	t *testing.T,
) {

	executor := newTestToolExecutor()

	executor.handlers["success"] = func(ctx context.Context) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "success",
		}, nil
	}

	executor.handlers["failure"] = func(ctx context.Context) (*llm.ToolResult, error) {
		return nil, fmt.Errorf("database unavailable")
	}

	service := &ChatService{
		executor: executor,
	}

	calls := []llm.ToolCall{
		{
			Function: llm.ToolFunctionCall{
				Name: "success",
			},
		},
		{
			Function: llm.ToolFunctionCall{
				Name: "failure",
			},
		},
	}

	results, err := service.executeTools(
		context.Background(),
		calls,
	)

	if err != nil {
		t.Fatalf(
			"unexpected executeTools error: %v",
			err,
		)
	}

	if len(results) != 2 {
		t.Fatalf(
			"expected 2 results, got %d",
			len(results),
		)
	}

	if results[0].Err != nil {
		t.Fatalf(
			"unexpected error for successful tool: %v",
			results[0].Err,
		)
	}

	if results[1].Err == nil {
		t.Fatal(
			"expected individual tool error",
		)
	}
}

func TestChatService_AppendToolResult_Error(
	t *testing.T,
) {

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	call := llm.ToolCall{
		Function: llm.ToolFunctionCall{
			Name: "database_query",
		},
	}

	err := (&ChatService{}).appendToolResult(
		conv,
		ToolExecutionResult{
			Call: call,
			Err:  fmt.Errorf("database unavailable"),
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	messages := conv.Messages()

	if len(messages) != 2 {
		t.Fatalf(
			"expected system + tool message, got %d messages",
			len(messages),
		)
	}

	message := messages[1]

	if message.Role != llm.RoleTool {
		t.Fatalf(
			"expected tool role, got %q",
			message.Role,
		)
	}

	if message.ToolName != "database_query" {
		t.Fatalf(
			"expected tool name database_query, got %q",
			message.ToolName,
		)
	}

	if message.Content == "" {
		t.Fatal("expected tool error content")
	}
}

type testChatClient struct {
	mu sync.Mutex

	requests  []llm.ChatRequest
	responses []llm.ChatResponse
}

func (c *testChatClient) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests = append(
		c.requests,
		req,
	)

	if len(c.responses) == 0 {
		return llm.ChatResponse{}, fmt.Errorf(
			"no test response available",
		)
	}

	response := c.responses[0]
	c.responses = c.responses[1:]

	return response, nil
}

func (c *testChatClient) Stream(
	ctx context.Context,
	req llm.ChatRequest,
) <-chan llm.StreamResult {

	ch := make(chan llm.StreamResult, 1)

	ch <- llm.StreamResult{
		Err: fmt.Errorf("stream not implemented in test client"),
	}

	close(ch)

	return ch
}

func TestChatService_ExecuteAgentLoop_MultipleToolCalls(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["tool-a"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {

		return &llm.ToolResult{
			Content: "result-a",
		}, nil
	}

	executor.handlers["tool-b"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {

		return &llm.ToolResult{
			Content: "result-b",
		}, nil
	}

	client := &testChatClient{
		responses: []llm.ChatResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							Function: llm.ToolFunctionCall{
								Name:      "tool-a",
								Arguments: json.RawMessage(`{}`),
							},
						},
						{
							Function: llm.ToolFunctionCall{
								Name:      "tool-b",
								Arguments: json.RawMessage(`{}`),
							},
						},
					},
				},
			},
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "Both tools completed successfully.",
				},
			},
		},
	}

	service := &ChatService{
		client:   client,
		executor: executor,
		core:     core.New(events.NopObserver{}),
	}

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"What do the tools say?",
	)

	response, err := service.executeAgentLoop(
		context.Background(),
		conv,
	)
	if err != nil {
		t.Fatal(err)
	}

	if response != "Both tools completed successfully." {
		t.Fatalf(
			"unexpected final response: %q",
			response,
		)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if len(client.requests) != 2 {
		t.Fatalf(
			"expected 2 LLM requests, got %d",
			len(client.requests),
		)
	}

	secondRequest := client.requests[1]

	if len(secondRequest.Messages) != 5 {
		t.Fatalf(
			"expected 5 messages in second request, got %d",
			len(secondRequest.Messages),
		)
	}

	// system
	if secondRequest.Messages[0].Role != llm.RoleSystem {
		t.Fatalf(
			"message 0: expected system, got %q",
			secondRequest.Messages[0].Role,
		)
	}

	// user
	if secondRequest.Messages[1].Role != llm.RoleUser {
		t.Fatalf(
			"message 1: expected user, got %q",
			secondRequest.Messages[1].Role,
		)
	}

	// assistant tool call
	assistant := secondRequest.Messages[2]

	if assistant.Role != llm.RoleAssistant {
		t.Fatalf(
			"message 2: expected assistant, got %q",
			assistant.Role,
		)
	}

	if len(assistant.ToolCalls) != 2 {
		t.Fatalf(
			"expected 2 tool calls, got %d",
			len(assistant.ToolCalls),
		)
	}

	if assistant.ToolCalls[0].Function.Name != "tool-a" {
		t.Fatalf(
			"expected first tool call to be tool-a, got %q",
			assistant.ToolCalls[0].Function.Name,
		)
	}

	if assistant.ToolCalls[1].Function.Name != "tool-b" {
		t.Fatalf(
			"expected second tool call to be tool-b, got %q",
			assistant.ToolCalls[1].Function.Name,
		)
	}

	// tool-a result
	toolA := secondRequest.Messages[3]

	if toolA.Role != llm.RoleTool {
		t.Fatalf(
			"message 3: expected tool, got %q",
			toolA.Role,
		)
	}

	if toolA.ToolName != "tool-a" {
		t.Fatalf(
			"message 3: expected tool-a, got %q",
			toolA.ToolName,
		)
	}

	if toolA.Content != "result-a" {
		t.Fatalf(
			"message 3: expected result-a, got %q",
			toolA.Content,
		)
	}

	// tool-b result
	toolB := secondRequest.Messages[4]

	if toolB.Role != llm.RoleTool {
		t.Fatalf(
			"message 4: expected tool, got %q",
			toolB.Role,
		)
	}

	if toolB.ToolName != "tool-b" {
		t.Fatalf(
			"message 4: expected tool-b, got %q",
			toolB.ToolName,
		)
	}

	if toolB.Content != "result-b" {
		t.Fatalf(
			"message 4: expected result-b, got %q",
			toolB.Content,
		)
	}
}
