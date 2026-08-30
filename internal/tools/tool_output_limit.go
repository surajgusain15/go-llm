package tools

import (
	"context"

	"go-llm/internal/llm"
	"go-llm/internal/llmutil"
)

func ToolOutputLimit(
	maxBytes int,
) ToolMiddleware {

	return func(next Handler) Handler {

		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			result, err := next(
				ctx,
				invocation,
			)

			if err != nil || result == nil {
				return result, err
			}

			content, err := llmutil.ToolResultToString(
				result.Content,
			)
			if err != nil {
				return nil, err
			}

			if len(content) <= maxBytes {
				return result, nil
			}

			return &llm.ToolResult{
				Content: map[string]any{
					"truncated":      true,
					"original_bytes": len(content),
					"max_bytes":      maxBytes,
				},
				Metadata: result.Metadata,
			}, nil
		}
	}
}

func WithToolOutputLimit(
	maxBytes int,
) ExecutorOption {
	return func(e *Executor) {
		e.middlewares = append(
			e.middlewares,
			ToolOutputLimit(maxBytes),
		)
	}
}
