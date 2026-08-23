package builtin

import (
	"context"
	"encoding/json"

	"go-llm/internal/llm"

	"github.com/google/uuid"
)

type UUIDTool struct{}

func NewUUIDTool() *UUIDTool {
	return &UUIDTool{}
}

func (u *UUIDTool) Schema() llm.ToolDefinition {

	return llm.ToolDefinition{
		Type: llm.ToolTypeFunction,
		Function: llm.ToolFunction{
			Name:        "uuid",
			Description: "Generates a random UUID.",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (u *UUIDTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (any, error) {

	return uuid.NewString(), nil
}
