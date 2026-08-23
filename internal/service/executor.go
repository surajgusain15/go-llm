package service

import (
	"context"
	"encoding/json"

	"go-llm/internal/llm"
)

type ToolExecutor interface {
	Schemas() []llm.ToolDefinition

	Execute(
		ctx context.Context,
		name string,
		input json.RawMessage,
	) (*llm.ToolResult, error)
}
