package tools

import (
	"context"

	"go-llm/internal/llm"
)

type Handler func(
	ctx context.Context,
	invocation ToolInvocation,
) (*llm.ToolResult, error)

type Middleware func(Handler) Handler
