package tools

import (
	"context"
	"encoding/json"

	"go-llm/internal/llm"
)

type Tool interface {
	Schema() llm.ToolDefinition

	Execute(
		ctx context.Context,
		input json.RawMessage,
	) (any, error)
}
