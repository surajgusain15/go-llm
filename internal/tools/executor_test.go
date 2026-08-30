package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go-llm/internal/llm"
)

type blockingTestTool struct {
	name    string
	started chan struct{}
}

func (t *blockingTestTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        t.name,
			Description: "blocking test tool",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *blockingTestTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	close(t.started)

	<-ctx.Done()

	return nil, ctx.Err()
}

func TestExecutor_Execute_AppliesToolTimeout(
	t *testing.T,
) {
	registry := NewRegistry()

	started := make(chan struct{})

	registry.Register(
		&blockingTestTool{
			name:    "blocking",
			started: started,
		},
	)

	executor := NewExecutor(
		registry,
		nil,
		WithToolTimeout(50*time.Millisecond),
		WithToolOutputLimit(64*1024),
	)

	start := time.Now()

	_, err := executor.Execute(
		context.Background(),
		"blocking",
		json.RawMessage(`{}`),
	)

	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if !errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		t.Fatalf(
			"expected context.DeadlineExceeded, got %v",
			err,
		)
	}

	select {
	case <-started:
		// Tool actually executed.
	default:
		t.Fatal("tool did not execute")
	}

	if elapsed < 50*time.Millisecond {
		t.Fatalf(
			"executor returned before configured timeout: %v",
			elapsed,
		)
	}
}

type largeOutputTestTool struct{}

func (t *largeOutputTestTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "large_output",
			Description: "returns a large result",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *largeOutputTestTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {
	return &llm.ToolResult{
		Content: "this output is definitely larger than ten bytes",
	}, nil
}

func TestExecutor_Execute_AppliesToolOutputLimit(
	t *testing.T,
) {
	registry := NewRegistry()

	registry.Register(
		&largeOutputTestTool{},
	)

	executor := NewExecutor(
		registry,
		nil,
		WithToolOutputLimit(10),
	)

	result, err := executor.Execute(
		context.Background(),
		"large_output",
		json.RawMessage(`{}`),
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if result == nil {
		t.Fatal("expected tool result")
	}

	content, ok := result.Content.(map[string]any)
	if !ok {
		t.Fatalf(
			"expected truncation metadata object, got %T",
			result.Content,
		)
	}

	if content["truncated"] != true {
		t.Fatalf(
			"expected truncated=true, got %v",
			content["truncated"],
		)
	}

	if content["max_bytes"] != 10 {
		t.Fatalf(
			"expected max_bytes=10, got %v",
			content["max_bytes"],
		)
	}
}
