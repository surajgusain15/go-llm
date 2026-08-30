package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"go-llm/internal/conversation"
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
