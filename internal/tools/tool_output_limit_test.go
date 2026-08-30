package tools

import (
	"context"
	"errors"
	"testing"

	"go-llm/internal/llm"
)

func TestToolOutputLimit_UnderLimit(
	t *testing.T,
) {
	expected := "small result"

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return &llm.ToolResult{
				Content: expected,
			}, nil
		},
	)

	wrapped := ToolOutputLimit(100)(
		handler,
	)

	result, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if result.Content != expected {
		t.Fatalf(
			"expected %q, got %v",
			expected,
			result.Content,
		)
	}
}

func TestToolOutputLimit_OverLimit(
	t *testing.T,
) {
	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return &llm.ToolResult{
				Content: "this is a very large tool response",
			}, nil
		},
	)

	wrapped := ToolOutputLimit(10)(
		handler,
	)

	result, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
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

func TestToolOutputLimit_ToolError(
	t *testing.T,
) {
	expectedErr := errors.New("database unavailable")

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return nil, expectedErr
		},
	)

	wrapped := ToolOutputLimit(10)(
		handler,
	)

	result, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	if result != nil {
		t.Fatalf(
			"expected nil result, got %v",
			result,
		)
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected original error, got %v",
			err,
		)
	}
}
