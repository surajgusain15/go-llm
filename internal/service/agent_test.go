package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
		call,
		nil,
		fmt.Errorf("database unavailable"),
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

func TestChatService_ExecuteAgentLoop_RecoversFromToolError(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["successful_tool"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "success-result",
		}, nil
	}

	executor.handlers["failing_tool"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return nil, fmt.Errorf("database unavailable")
	}

	client := &testChatClient{
		responses: []llm.ChatResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							Function: llm.ToolFunctionCall{
								Name:      "successful_tool",
								Arguments: json.RawMessage(`{}`),
							},
						},
						{
							Function: llm.ToolFunctionCall{
								Name:      "failing_tool",
								Arguments: json.RawMessage(`{}`),
							},
						},
					},
				},
			},
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "I could not complete the second operation because the database is unavailable.",
				},
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"Run both operations.",
	)

	response, err := service.executeAgentLoop(
		context.Background(),
		conv,
	)
	if err != nil {
		t.Fatal(err)
	}

	expected :=
		"I could not complete the second operation because the database is unavailable."

	if response != expected {
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

	second := client.requests[1]

	// system + user + assistant tool calls + 2 tool results
	if len(second.Messages) != 5 {
		t.Fatalf(
			"expected 5 messages in second request, got %d",
			len(second.Messages),
		)
	}

	success := second.Messages[3]

	if success.Role != llm.RoleTool {
		t.Fatalf(
			"message 3: expected tool role, got %q",
			success.Role,
		)
	}

	if success.ToolName != "successful_tool" {
		t.Fatalf(
			"message 3: expected successful_tool, got %q",
			success.ToolName,
		)
	}

	if success.Content != "success-result" {
		t.Fatalf(
			"message 3: expected success-result, got %q",
			success.Content,
		)
	}

	failure := second.Messages[4]

	if failure.Role != llm.RoleTool {
		t.Fatalf(
			"message 4: expected tool role, got %q",
			failure.Role,
		)
	}

	if failure.ToolName != "failing_tool" {
		t.Fatalf(
			"message 4: expected failing_tool, got %q",
			failure.ToolName,
		)
	}

	if failure.Content == "" {
		t.Fatal(
			"expected failing tool to produce error content",
		)
	}
}

func TestChatService_ExecuteStreamingAgentLoop_ToolCall(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_schema"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "schema-result",
		}, nil
	}

	client := &testStreamingClient{
		streams: []testStream{
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
							ToolCalls: []llm.ToolCall{
								{
									Function: llm.ToolFunctionCall{
										Name:      "database_schema",
										Arguments: json.RawMessage(`{}`),
									},
								},
							},
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Done: true,
					},
				},
			},
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role:    llm.RoleAssistant,
							Content: "The schema is available.",
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Done: true,
					},
				},
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"What is the database schema?",
	)

	stream := service.executeStreamingAgentLoop(
		context.Background(),
		conv,
	)

	var chunks []llm.StreamResult

	for result := range stream {
		if result.Err != nil {
			t.Fatal(result.Err)
		}

		chunks = append(
			chunks,
			result,
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

	if len(conv.Messages()) != 5 {
		t.Fatalf(
			"expected 5 conversation messages, got %d",
			len(conv.Messages()),
		)
	}

	messages := conv.Messages()

	if messages[0].Role != llm.RoleSystem {
		t.Fatalf(
			"message 0: expected system, got %q",
			messages[0].Role,
		)
	}

	if messages[1].Role != llm.RoleUser {
		t.Fatalf(
			"message 1: expected user, got %q",
			messages[1].Role,
		)
	}

	if messages[2].Role != llm.RoleAssistant {
		t.Fatalf(
			"message 2: expected assistant, got %q",
			messages[2].Role,
		)
	}

	if len(messages[2].ToolCalls) != 1 {
		t.Fatalf(
			"expected 1 tool call, got %d",
			len(messages[2].ToolCalls),
		)
	}

	if messages[3].Role != llm.RoleTool {
		t.Fatalf(
			"message 3: expected tool, got %q",
			messages[3].Role,
		)
	}

	if messages[3].ToolName != "database_schema" {
		t.Fatalf(
			"expected database_schema, got %q",
			messages[3].ToolName,
		)
	}

	if messages[3].Content != "schema-result" {
		t.Fatalf(
			"expected schema-result, got %q",
			messages[3].Content,
		)
	}

	final := messages[4]

	if final.Role != llm.RoleAssistant {
		t.Fatalf(
			"message 4: expected assistant, got %q",
			final.Role,
		)
	}

	if final.Content != "The schema is available." {
		t.Fatalf(
			"message 4: expected final response, got %q",
			final.Content,
		)
	}

	_ = chunks
}

type testStream []llm.StreamResult

type testStreamingClient struct {
	mu       sync.Mutex
	requests []llm.ChatRequest
	streams  []testStream
}

func (c *testStreamingClient) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {
	return llm.ChatResponse{}, fmt.Errorf(
		"Chat not implemented in streaming test client",
	)
}

func (c *testStreamingClient) Stream(
	ctx context.Context,
	req llm.ChatRequest,
) <-chan llm.StreamResult {

	c.mu.Lock()

	c.requests = append(
		c.requests,
		req,
	)

	if len(c.streams) == 0 {
		c.mu.Unlock()

		ch := make(chan llm.StreamResult, 1)
		ch <- llm.StreamResult{
			Err: fmt.Errorf("no test stream available"),
		}
		close(ch)

		return ch
	}

	stream := c.streams[0]
	c.streams = c.streams[1:]

	c.mu.Unlock()

	ch := make(chan llm.StreamResult, len(stream))

	for _, result := range stream {
		ch <- result
	}

	close(ch)

	return ch
}

func TestChatService_ExecuteStreamingAgentLoop_PreservesContentBeforeToolCall(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_schema"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "schema-result",
		}, nil
	}

	client := &testStreamingClient{
		streams: []testStream{
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role:    llm.RoleAssistant,
							Content: "I'll check the database schema.",
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
							ToolCalls: []llm.ToolCall{
								{
									Function: llm.ToolFunctionCall{
										Name:      "database_schema",
										Arguments: json.RawMessage(`{}`),
									},
								},
							},
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Done: true,
					},
				},
			},
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role:    llm.RoleAssistant,
							Content: "Here is the schema.",
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Done: true,
					},
				},
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"What is the database schema?",
	)

	stream := service.executeStreamingAgentLoop(
		context.Background(),
		conv,
	)

	for result := range stream {
		if result.Err != nil {
			t.Fatal(result.Err)
		}
	}

	messages := conv.Messages()

	if len(messages) != 5 {
		t.Fatalf(
			"expected 5 conversation messages, got %d",
			len(messages),
		)
	}

	assistantToolMessage := messages[2]

	if assistantToolMessage.Role != llm.RoleAssistant {
		t.Fatalf(
			"expected assistant message, got %q",
			assistantToolMessage.Role,
		)
	}

	if assistantToolMessage.Content != "I'll check the database schema." {
		t.Fatalf(
			"expected assistant content to be preserved, got %q",
			assistantToolMessage.Content,
		)
	}

	if len(assistantToolMessage.ToolCalls) != 1 {
		t.Fatalf(
			"expected 1 tool call, got %d",
			len(assistantToolMessage.ToolCalls),
		)
	}
}

func TestChatService_ExecuteStreamingAgentLoop_MultipleToolCalls(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_schema"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "schema-result",
		}, nil
	}

	executor.handlers["database_query"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "query-result",
		}, nil
	}

	client := &testStreamingClient{
		streams: []testStream{
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
							ToolCalls: []llm.ToolCall{
								{
									Function: llm.ToolFunctionCall{
										Name:      "database_schema",
										Arguments: json.RawMessage(`{}`),
									},
								},
								{
									Function: llm.ToolFunctionCall{
										Name: "database_query",
										Arguments: json.RawMessage(
											`{"query":"SELECT 1"}`,
										),
									},
								},
							},
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
						},
						Done: true,
					},
				},
			},
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role:    llm.RoleAssistant,
							Content: "Done.",
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
						},
						Done: true,
					},
				},
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"Get the schema and run a query.",
	)

	stream := service.executeStreamingAgentLoop(
		context.Background(),
		conv,
	)

	for result := range stream {
		if result.Err != nil {
			t.Fatal(result.Err)
		}
	}

	messages := conv.Messages()

	// system
	// user
	// assistant with 2 tool calls
	// tool: database_schema
	// tool: database_query
	// final assistant
	if len(messages) != 6 {
		t.Fatalf(
			"expected 6 conversation messages, got %d",
			len(messages),
		)
	}

	assistant := messages[2]

	if assistant.Role != llm.RoleAssistant {
		t.Fatalf(
			"expected assistant message, got %q",
			assistant.Role,
		)
	}

	if len(assistant.ToolCalls) != 2 {
		t.Fatalf(
			"expected 2 tool calls, got %d",
			len(assistant.ToolCalls),
		)
	}

	if assistant.ToolCalls[0].Function.Name != "database_schema" {
		t.Fatalf(
			"expected first tool call database_schema, got %q",
			assistant.ToolCalls[0].Function.Name,
		)
	}

	if assistant.ToolCalls[1].Function.Name != "database_query" {
		t.Fatalf(
			"expected second tool call database_query, got %q",
			assistant.ToolCalls[1].Function.Name,
		)
	}

	firstTool := messages[3]

	if firstTool.Role != llm.RoleTool {
		t.Fatalf(
			"expected first tool message, got %q",
			firstTool.Role,
		)
	}

	if firstTool.ToolName != "database_schema" {
		t.Fatalf(
			"expected database_schema, got %q",
			firstTool.ToolName,
		)
	}

	if firstTool.Content != "schema-result" {
		t.Fatalf(
			"expected schema-result, got %q",
			firstTool.Content,
		)
	}

	secondTool := messages[4]

	if secondTool.Role != llm.RoleTool {
		t.Fatalf(
			"expected second tool message, got %q",
			secondTool.Role,
		)
	}

	if secondTool.ToolName != "database_query" {
		t.Fatalf(
			"expected database_query, got %q",
			secondTool.ToolName,
		)
	}

	if secondTool.Content != "query-result" {
		t.Fatalf(
			"expected query-result, got %q",
			secondTool.Content,
		)
	}

	final := messages[5]

	if final.Role != llm.RoleAssistant {
		t.Fatalf(
			"expected final assistant message, got %q",
			final.Role,
		)
	}

	if final.Content != "Done." {
		t.Fatalf(
			"expected final response Done., got %q",
			final.Content,
		)
	}
}

func TestChatService_ExecuteStreamingAgentLoop_ToolError(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_query"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return nil, fmt.Errorf("database unavailable")
	}

	client := &testStreamingClient{
		streams: []testStream{
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
							ToolCalls: []llm.ToolCall{
								{
									Function: llm.ToolFunctionCall{
										Name: "database_query",
										Arguments: json.RawMessage(
											`{"query":"SELECT 1"}`,
										),
									},
								},
							},
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
						},
						Done: true,
					},
				},
			},
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role:    llm.RoleAssistant,
							Content: "The database is currently unavailable.",
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
						},
						Done: true,
					},
				},
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"Run a database query.",
	)

	stream := service.executeStreamingAgentLoop(
		context.Background(),
		conv,
	)

	for result := range stream {
		if result.Err != nil {
			t.Fatal(result.Err)
		}
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != 2 {
		t.Fatalf(
			"expected 2 LLM requests, got %d",
			requestCount,
		)
	}

	messages := conv.Messages()

	// system
	// user
	// assistant with tool call
	// tool error
	// final assistant
	if len(messages) != 5 {
		t.Fatalf(
			"expected 5 conversation messages, got %d",
			len(messages),
		)
	}

	if messages[2].Role != llm.RoleAssistant {
		t.Fatalf(
			"expected message 2 to be assistant, got %q",
			messages[2].Role,
		)
	}

	if len(messages[2].ToolCalls) != 1 {
		t.Fatalf(
			"expected 1 tool call, got %d",
			len(messages[2].ToolCalls),
		)
	}

	if messages[3].Role != llm.RoleTool {
		t.Fatalf(
			"expected message 3 to be tool, got %q",
			messages[3].Role,
		)
	}

	if messages[3].ToolName != "database_query" {
		t.Fatalf(
			"expected database_query, got %q",
			messages[3].ToolName,
		)
	}

	if messages[3].Content == "" {
		t.Fatal(
			"expected tool error content",
		)
	}

	if messages[4].Role != llm.RoleAssistant {
		t.Fatalf(
			"expected message 4 to be assistant, got %q",
			messages[4].Role,
		)
	}

	if messages[4].Content != "The database is currently unavailable." {
		t.Fatalf(
			"unexpected final response: %q",
			messages[4].Content,
		)
	}
}

type blockingStreamingClient struct {
	mu       sync.Mutex
	requests []llm.ChatRequest
}

func (c *blockingStreamingClient) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {
	return llm.ChatResponse{}, fmt.Errorf("Chat should not be called")
}

func (c *blockingStreamingClient) Stream(
	ctx context.Context,
	req llm.ChatRequest,
) <-chan llm.StreamResult {

	c.mu.Lock()
	c.requests = append(c.requests, req)
	c.mu.Unlock()

	ch := make(chan llm.StreamResult)

	go func() {
		defer close(ch)

		<-ctx.Done()

		ch <- llm.StreamResult{
			Err: ctx.Err(),
		}
	}()

	return ch
}

func TestChatService_ExecuteStreamingAgentLoop_ContextCancellation(
	t *testing.T,
) {
	client := &blockingStreamingClient{}

	executor := newTestToolExecutor()

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"Do something.",
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	stream := service.executeStreamingAgentLoop(
		ctx,
		conv,
	)

	// Give the streaming goroutine a chance to enter
	// the blocking client.
	time.Sleep(10 * time.Millisecond)

	cancel()

	done := make(chan error, 1)

	go func() {
		var streamErr error

		for result := range stream {
			if result.Err != nil {
				streamErr = result.Err
			}
		}

		done <- streamErr
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"streaming agent loop did not terminate after cancellation",
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != 1 {
		t.Fatalf(
			"expected 1 LLM request, got %d",
			requestCount,
		)
	}
}

type blockingToolStreamingClient struct {
	mu       sync.Mutex
	requests []llm.ChatRequest
}

func (c *blockingToolStreamingClient) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {
	return llm.ChatResponse{}, fmt.Errorf(
		"Chat should not be called",
	)
}

func (c *blockingToolStreamingClient) Stream(
	ctx context.Context,
	req llm.ChatRequest,
) <-chan llm.StreamResult {

	c.mu.Lock()
	c.requests = append(c.requests, req)
	c.mu.Unlock()

	out := make(chan llm.StreamResult, 2)

	out <- llm.StreamResult{
		Chunk: llm.StreamChunk{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						Function: llm.ToolFunctionCall{
							Name: "database_query",
							Arguments: json.RawMessage(
								`{"query":"SELECT 1"}`,
							),
						},
					},
				},
			},
		},
	}

	out <- llm.StreamResult{
		Chunk: llm.StreamChunk{
			Message: llm.Message{
				Role: llm.RoleAssistant,
			},
			Done: true,
		},
	}

	close(out)

	return out
}

func TestChatService_ExecuteStreamingAgentLoop_CancelDuringToolExecution(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	toolStarted := make(chan struct{})

	executor.handlers["database_query"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		close(toolStarted)

		<-ctx.Done()

		return nil, ctx.Err()
	}

	client := &blockingToolStreamingClient{}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	conv.AddUserMessage(
		"Run a database query.",
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	stream := service.executeStreamingAgentLoop(
		ctx,
		conv,
	)

	streamErrCh := make(chan error, 1)

	go func() {
		var streamErr error

		for result := range stream {
			if result.Err != nil {
				streamErr = result.Err
			}
		}

		streamErrCh <- streamErr
	}()

	select {
	case <-toolStarted:
	case <-time.After(time.Second):
		t.Fatal("tool did not start")
	}

	cancel()

	select {
	case err := <-streamErrCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"streaming agent loop did not terminate after cancellation",
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != 1 {
		t.Fatalf(
			"expected exactly 1 LLM request, got %d",
			requestCount,
		)
	}

	messages := conv.Messages()

	// system
	// user
	// assistant with tool call
	if len(messages) != 3 {
		t.Fatalf(
			"expected 3 conversation messages, got %d",
			len(messages),
		)
	}

	if messages[2].Role != llm.RoleAssistant {
		t.Fatalf(
			"expected assistant message, got %q",
			messages[2].Role,
		)
	}
}

func TestChatService_ExecuteAgentLoop_MaxIterations(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_query"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "tool result",
		}, nil
	}

	client := &testChatClient{
		responses: make([]llm.ChatResponse, maxAgentIterations),
	}

	for i := range maxAgentIterations {
		client.responses[i] = llm.ChatResponse{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						Function: llm.ToolFunctionCall{
							Name: "database_query",
							Arguments: json.RawMessage(
								`{"query":"SELECT 1"}`,
							),
						},
					},
				},
			},
		}
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := service.NewConversation()

	_, err := service.executeAgentLoop(
		context.Background(),
		conv,
	)

	if err == nil {
		t.Fatal(
			"expected maximum iteration error",
		)
	}

	if err.Error() != "maximum agent iterations exceeded" {
		t.Fatalf(
			"expected maximum agent iterations exceeded, got %q",
			err,
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != maxAgentIterations {
		t.Fatalf(
			"expected %d LLM requests, got %d",
			maxAgentIterations,
			requestCount,
		)
	}
}

func TestChatService_ExecuteStreamingAgentLoop_MaxIterations(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_query"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "tool result",
		}, nil
	}

	streams := make([]testStream, maxAgentIterations)

	for i := range maxAgentIterations {
		streams[i] = testStream{
			{
				Chunk: llm.StreamChunk{
					Message: llm.Message{
						Role: llm.RoleAssistant,
						ToolCalls: []llm.ToolCall{
							{
								Function: llm.ToolFunctionCall{
									Name: "database_query",
									Arguments: json.RawMessage(
										`{"query":"SELECT 1"}`,
									),
								},
							},
						},
					},
				},
			},
			{
				Chunk: llm.StreamChunk{
					Message: llm.Message{
						Role: llm.RoleAssistant,
					},
					Done: true,
				},
			},
		}
	}

	client := &testStreamingClient{
		streams: streams,
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := service.NewConversation()

	stream := service.executeStreamingAgentLoop(
		context.Background(),
		conv,
	)

	var streamErr error

	for result := range stream {
		if result.Err != nil {
			streamErr = result.Err
		}
	}

	if streamErr == nil {
		t.Fatal(
			"expected maximum iteration error",
		)
	}

	if streamErr.Error() != "maximum agent iterations exceeded" {
		t.Fatalf(
			"expected maximum agent iterations exceeded, got %q",
			streamErr,
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != maxAgentIterations {
		t.Fatalf(
			"expected %d LLM requests, got %d",
			maxAgentIterations,
			requestCount,
		)
	}
}

func TestChatService_Chat(t *testing.T) {
	client := &testChatClient{
		responses: []llm.ChatResponse{
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "Hello from the assistant.",
				},
			},
		},
	}

	executor := newTestToolExecutor()

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := service.NewConversation()

	response, err := service.Chat(
		context.Background(),
		conv,
		"Hello",
	)
	if err != nil {
		t.Fatalf(
			"Chat returned error: %v",
			err,
		)
	}

	if response != "Hello from the assistant." {
		t.Fatalf(
			"expected assistant response %q, got %q",
			"Hello from the assistant.",
			response,
		)
	}

	messages := conv.Messages()

	// system
	// user
	// assistant
	if len(messages) != 3 {
		t.Fatalf(
			"expected 3 conversation messages, got %d",
			len(messages),
		)
	}

	if messages[0].Role != llm.RoleSystem {
		t.Fatalf(
			"expected first message to be system, got %q",
			messages[0].Role,
		)
	}

	if messages[1].Role != llm.RoleUser {
		t.Fatalf(
			"expected second message to be user, got %q",
			messages[1].Role,
		)
	}

	if messages[1].Content != "Hello" {
		t.Fatalf(
			"expected user message %q, got %q",
			"Hello",
			messages[1].Content,
		)
	}

	if messages[2].Role != llm.RoleAssistant {
		t.Fatalf(
			"expected third message to be assistant, got %q",
			messages[2].Role,
		)
	}

	if messages[2].Content != "Hello from the assistant." {
		t.Fatalf(
			"unexpected assistant content: %q",
			messages[2].Content,
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != 1 {
		t.Fatalf(
			"expected exactly 1 LLM request, got %d",
			requestCount,
		)
	}
}

func TestChatService_Stream(t *testing.T) {
	client := &testStreamingClient{
		streams: []testStream{
			{
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role:    llm.RoleAssistant,
							Content: "Hello ",
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role:    llm.RoleAssistant,
							Content: "from the assistant.",
						},
					},
				},
				{
					Chunk: llm.StreamChunk{
						Message: llm.Message{
							Role: llm.RoleAssistant,
						},
						Done: true,
					},
				},
			},
		},
	}

	executor := newTestToolExecutor()

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := service.NewConversation()

	stream := service.Stream(
		context.Background(),
		conv,
		"Hello",
	)

	var content strings.Builder

	for result := range stream {
		if result.Err != nil {
			t.Fatalf(
				"Stream returned error: %v",
				result.Err,
			)
		}

		content.WriteString(
			result.Chunk.Message.Content,
		)
	}

	if content.String() != "Hello from the assistant." {
		t.Fatalf(
			"expected streamed content %q, got %q",
			"Hello from the assistant.",
			content.String(),
		)
	}

	messages := conv.Messages()

	// system
	// user
	// assistant
	if len(messages) != 3 {
		t.Fatalf(
			"expected 3 conversation messages, got %d",
			len(messages),
		)
	}

	if messages[0].Role != llm.RoleSystem {
		t.Fatalf(
			"expected first message to be system, got %q",
			messages[0].Role,
		)
	}

	if messages[1].Role != llm.RoleUser {
		t.Fatalf(
			"expected second message to be user, got %q",
			messages[1].Role,
		)
	}

	if messages[1].Content != "Hello" {
		t.Fatalf(
			"expected user message %q, got %q",
			"Hello",
			messages[1].Content,
		)
	}

	if messages[2].Role != llm.RoleAssistant {
		t.Fatalf(
			"expected third message to be assistant, got %q",
			messages[2].Role,
		)
	}

	if messages[2].Content != "Hello from the assistant." {
		t.Fatalf(
			"expected final assistant content %q, got %q",
			"Hello from the assistant.",
			messages[2].Content,
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != 1 {
		t.Fatalf(
			"expected exactly 1 LLM request, got %d",
			requestCount,
		)
	}
}

func TestChatService_Chat_PreservesConversationHistory(
	t *testing.T,
) {
	client := &testChatClient{
		responses: []llm.ChatResponse{
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "First response",
				},
			},
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "Second response",
				},
			},
		},
	}

	executor := newTestToolExecutor()

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := service.NewConversation()

	response, err := service.Chat(
		context.Background(),
		conv,
		"First question",
	)
	if err != nil {
		t.Fatalf(
			"first Chat returned error: %v",
			err,
		)
	}

	if response != "First response" {
		t.Fatalf(
			"unexpected first response: %q",
			response,
		)
	}

	response, err = service.Chat(
		context.Background(),
		conv,
		"Second question",
	)
	if err != nil {
		t.Fatalf(
			"second Chat returned error: %v",
			err,
		)
	}

	if response != "Second response" {
		t.Fatalf(
			"unexpected second response: %q",
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

	if len(secondRequest.Messages) != 4 {
		t.Fatalf(
			"expected 4 messages in second request, got %d",
			len(secondRequest.Messages),
		)
	}

	expected := []struct {
		role    llm.Role
		content string
	}{
		{llm.RoleSystem, DefaultSystemPrompt},
		{llm.RoleUser, "First question"},
		{llm.RoleAssistant, "First response"},
		{llm.RoleUser, "Second question"},
	}

	for i, want := range expected {
		got := secondRequest.Messages[i]

		if got.Role != want.role {
			t.Fatalf(
				"message %d: expected role %q, got %q",
				i,
				want.role,
				got.Role,
			)
		}

		if got.Content != want.content {
			t.Fatalf(
				"message %d: expected content %q, got %q",
				i,
				want.content,
				got.Content,
			)
		}
	}
}

func TestChatService_Chat_PreservesToolConversationHistory(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_schema"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: "tables: users, transactions",
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
								Name:      "database_schema",
								Arguments: json.RawMessage(`{}`),
							},
						},
					},
				},
			},
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "The database contains users and transactions.",
				},
			},
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "The transactions table contains transaction records.",
				},
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := service.NewConversation()

	_, err := service.Chat(
		context.Background(),
		conv,
		"What tables are in the database?",
	)
	if err != nil {
		t.Fatalf(
			"first Chat returned error: %v",
			err,
		)
	}

	response, err := service.Chat(
		context.Background(),
		conv,
		"Which table contains transactions?",
	)
	if err != nil {
		t.Fatalf(
			"second Chat returned error: %v",
			err,
		)
	}

	if response != "The transactions table contains transaction records." {
		t.Fatalf(
			"unexpected response: %q",
			response,
		)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if len(client.requests) != 3 {
		t.Fatalf(
			"expected 3 LLM requests, got %d",
			len(client.requests),
		)
	}

	secondUserRequest := client.requests[2]

	if len(secondUserRequest.Messages) != 6 {
		t.Fatalf(
			"expected 6 messages in final LLM request, got %d",
			len(secondUserRequest.Messages),
		)
	}

	expected := []struct {
		role    llm.Role
		content string
	}{
		{llm.RoleSystem, DefaultSystemPrompt},
		{llm.RoleUser, "What tables are in the database?"},
		{llm.RoleAssistant, ""},
		{llm.RoleTool, "tables: users, transactions"},
		{
			llm.RoleAssistant,
			"The database contains users and transactions.",
		},
		{llm.RoleUser, "Which table contains transactions?"},
	}

	for i, want := range expected {
		got := secondUserRequest.Messages[i]

		if got.Role != want.role {
			t.Fatalf(
				"message %d: expected role %q, got %q",
				i,
				want.role,
				got.Role,
			)
		}

		if got.Content != want.content {
			t.Fatalf(
				"message %d: expected content %q, got %q",
				i,
				want.content,
				got.Content,
			)
		}
	}

	toolCall := secondUserRequest.Messages[2].ToolCalls

	if len(toolCall) != 1 {
		t.Fatalf(
			"expected 1 tool call, got %d",
			len(toolCall),
		)
	}

	if toolCall[0].Function.Name != "database_schema" {
		t.Fatalf(
			"expected database_schema tool call, got %q",
			toolCall[0].Function.Name,
		)
	}

	if secondUserRequest.Messages[3].ToolName != "database_schema" {
		t.Fatalf(
			"expected tool name database_schema, got %q",
			secondUserRequest.Messages[3].ToolName,
		)
	}
}

func TestChatService_Chat_ToolErrorPreservesConversation(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	executor.handlers["database_schema"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return nil, errors.New("database unavailable")
	}

	client := &testChatClient{
		responses: []llm.ChatResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							Function: llm.ToolFunctionCall{
								Name:      "database_schema",
								Arguments: json.RawMessage(`{}`),
							},
						},
					},
				},
			},
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "I couldn't access the database because it is unavailable.",
				},
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	conv := service.NewConversation()

	response, err := service.Chat(
		context.Background(),
		conv,
		"What tables are in the database?",
	)

	if err != nil {
		t.Fatalf(
			"Chat returned error: %v",
			err,
		)
	}

	if response != "I couldn't access the database because it is unavailable." {
		t.Fatalf(
			"unexpected response: %q",
			response,
		)
	}

	messages := conv.Messages()

	// system
	// user
	// assistant with tool call
	// tool error
	// assistant final response
	if len(messages) != 5 {
		t.Fatalf(
			"expected 5 conversation messages, got %d",
			len(messages),
		)
	}

	if messages[0].Role != llm.RoleSystem {
		t.Fatalf(
			"expected message 0 to be system, got %q",
			messages[0].Role,
		)
	}

	if messages[1].Role != llm.RoleUser {
		t.Fatalf(
			"expected message 1 to be user, got %q",
			messages[1].Role,
		)
	}

	if messages[1].Content != "What tables are in the database?" {
		t.Fatalf(
			"unexpected user message: %q",
			messages[1].Content,
		)
	}

	// Assistant tool call.
	if messages[2].Role != llm.RoleAssistant {
		t.Fatalf(
			"expected message 2 to be assistant, got %q",
			messages[2].Role,
		)
	}

	if len(messages[2].ToolCalls) != 1 {
		t.Fatalf(
			"expected 1 tool call, got %d",
			len(messages[2].ToolCalls),
		)
	}

	if messages[2].ToolCalls[0].Function.Name != "database_schema" {
		t.Fatalf(
			"expected database_schema tool call, got %q",
			messages[2].ToolCalls[0].Function.Name,
		)
	}

	// Tool error is returned to the LLM as a tool message.
	if messages[3].Role != llm.RoleTool {
		t.Fatalf(
			"expected message 3 to be tool, got %q",
			messages[3].Role,
		)
	}

	if messages[3].ToolName != "database_schema" {
		t.Fatalf(
			"expected tool name database_schema, got %q",
			messages[3].ToolName,
		)
	}

	if !strings.Contains(
		messages[3].Content,
		"database unavailable",
	) {
		t.Fatalf(
			"expected tool error content to contain %q, got %q",
			"database unavailable",
			messages[3].Content,
		)
	}

	// Final assistant response after the tool error.
	if messages[4].Role != llm.RoleAssistant {
		t.Fatalf(
			"expected message 4 to be assistant, got %q",
			messages[4].Role,
		)
	}

	if messages[4].Content != "I couldn't access the database because it is unavailable." {
		t.Fatalf(
			"unexpected final assistant content: %q",
			messages[4].Content,
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != 2 {
		t.Fatalf(
			"expected 2 LLM requests, got %d",
			requestCount,
		)
	}
}

type blockingLLMStreamClient struct {
	mu sync.Mutex

	requests []llm.ChatRequest

	firstChunk llm.StreamResult
}

func (c *blockingLLMStreamClient) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {
	return llm.ChatResponse{}, fmt.Errorf(
		"Chat should not be called",
	)
}

func (c *blockingLLMStreamClient) Stream(
	ctx context.Context,
	req llm.ChatRequest,
) <-chan llm.StreamResult {

	c.mu.Lock()
	c.requests = append(
		c.requests,
		req,
	)
	c.mu.Unlock()

	out := make(chan llm.StreamResult)

	go func() {
		defer close(out)

		// Send one chunk immediately.
		out <- c.firstChunk

		// Simulate an LLM connection that remains open.
		// It must terminate when the request context is cancelled.
		<-ctx.Done()
	}()

	return out
}

func TestChatService_ExecuteStreamingAgentLoop_CancelDuringLLMStream(
	t *testing.T,
) {
	client := &blockingLLMStreamClient{
		firstChunk: llm.StreamResult{
			Chunk: llm.StreamChunk{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "Thinking...",
				},
			},
		},
	}

	service := NewChatService(
		client,
		newTestToolExecutor(),
		nil,
	)

	conv := conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	stream := service.executeStreamingAgentLoop(
		ctx,
		conv,
	)

	// The first token must arrive.
	select {
	case result := <-stream:
		if result.Err != nil {
			t.Fatalf(
				"unexpected initial stream error: %v",
				result.Err,
			)
		}

		if result.Chunk.Message.Content != "Thinking..." {
			t.Fatalf(
				"unexpected first chunk: %q",
				result.Chunk.Message.Content,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("LLM stream did not start")
	}

	// Cancel while the LLM stream is still active.
	cancel()

	done := make(chan error, 1)

	go func() {
		var streamErr error

		for result := range stream {
			if result.Err != nil {
				streamErr = result.Err
			}
		}

		done <- streamErr
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"streaming agent loop did not terminate after cancellation",
		)
	}

	client.mu.Lock()
	requestCount := len(client.requests)
	client.mu.Unlock()

	if requestCount != 1 {
		t.Fatalf(
			"expected exactly 1 LLM request, got %d",
			requestCount,
		)
	}

	messages := conv.Messages()

	// No final assistant message should be committed because
	// the stream never completed.
	if len(messages) != 1 {
		t.Fatalf(
			"expected only system message, got %d messages",
			len(messages),
		)
	}

	if messages[0].Role != llm.RoleSystem {
		t.Fatalf(
			"expected system message, got %q",
			messages[0].Role,
		)
	}
}

func TestLimitToolOutput_UnderLimit(
	t *testing.T,
) {
	content := "small result"

	got := limitToolOutput(
		content,
		100,
	)

	if got != content {
		t.Fatalf(
			"expected unchanged content, got %q",
			got,
		)
	}
}

func TestLimitToolOutput_OverLimit(
	t *testing.T,
) {
	content := strings.Repeat("x", 100)

	got := limitToolOutput(
		content,
		50,
	)

	if len(got) != 50 {
		t.Fatalf(
			"expected exactly 50 bytes, got %d",
			len(got),
		)
	}

	if !strings.Contains(
		got,
		"[tool output truncated:",
	) {
		t.Fatalf(
			"expected truncation marker, got %q",
			got,
		)
	}
}

func TestLimitToolOutput_Unicode(
	t *testing.T,
) {
	content := strings.Repeat("😀", 100)

	got := limitToolOutput(
		content,
		50,
	)

	if len(got) != 48 {
		t.Fatalf(
			"expected exactly 48 bytes, got %d",
			len(got),
		)
	}
}

func TestChatService_Chat_ToolOutputIsLimitedBeforeNextLLMRequest(
	t *testing.T,
) {
	executor := newTestToolExecutor()

	largeOutput := strings.Repeat("x", 100)

	executor.handlers["database_query"] = func(
		ctx context.Context,
	) (*llm.ToolResult, error) {
		return &llm.ToolResult{
			Content: largeOutput,
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
								Name: "database_query",
								Arguments: json.RawMessage(
									`{"query":"SELECT 1"}`,
								),
							},
						},
					},
				},
				Done: true,
			},
			{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "The query returned results.",
				},
				Done: true,
			},
		},
	}

	service := NewChatService(
		client,
		executor,
		nil,
	)

	// Keep this test fast and deterministic.
	service.toolOutputLimit = 50

	conv := service.NewConversation()

	_, err := service.Chat(
		context.Background(),
		conv,
		"Run a database query.",
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if len(client.requests) != 2 {
		t.Fatalf(
			"expected 2 LLM requests, got %d",
			len(client.requests),
		)
	}

	secondRequest := client.requests[1]

	var toolMessage *llm.Message

	for i := range secondRequest.Messages {
		message := &secondRequest.Messages[i]

		if message.Role == llm.RoleTool {
			toolMessage = message
			break
		}
	}

	if toolMessage == nil {
		t.Fatal("expected tool message in second LLM request")
	}

	if len(toolMessage.Content) > service.toolOutputLimit {
		t.Fatalf(
			"tool output exceeded limit: got %d bytes, limit %d",
			len(toolMessage.Content),
			service.toolOutputLimit,
		)
	}

	if !strings.Contains(
		toolMessage.Content,
		"[tool output truncated:",
	) {
		t.Fatalf(
			"expected truncation marker, got %q",
			toolMessage.Content,
		)
	}

	if strings.Contains(
		toolMessage.Content,
		largeOutput,
	) {
		t.Fatal(
			"full oversized tool output leaked into LLM request",
		)
	}
}
