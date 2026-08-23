package builtin

import (
	"context"
	"encoding/json"
	"time"

	"go-llm/internal/llm"
)

type TimeTool struct{}

func NewTimeTool() *TimeTool {
	return &TimeTool{}
}

func (t *TimeTool) Schema() llm.ToolDefinition {

	return llm.ToolDefinition{
		Type: llm.ToolTypeFunction,
		Function: llm.ToolFunction{
			Name:        "current_time",
			Description: "Returns the current local time.",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *TimeTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	return &llm.ToolResult{
		Content: time.Now().Format(time.RFC3339),
	}, nil
}
