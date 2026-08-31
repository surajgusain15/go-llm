package tools

import (
	"context"
	"encoding/json"
	"errors"

	"go-llm/internal/llm"
)

type Tool interface {
	Schema() llm.ToolDefinition

	Execute(
		ctx context.Context,
		input json.RawMessage,
	) (*llm.ToolResult, error)
}

func isTransientToolError(err error) bool {
	return errors.Is(err, ErrToolUnavailable) ||
		errors.Is(err, ErrToolRateLimited) ||
		errors.Is(err, ErrToolTemporary)
}
