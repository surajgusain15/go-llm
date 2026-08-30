package tools

import (
	"context"
	"time"

	"go-llm/internal/llm"
)

func ToolTimeout(
	timeout time.Duration,
) ToolMiddleware {

	return func(next Handler) Handler {

		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			toolCtx, cancel := context.WithTimeout(
				ctx,
				timeout,
			)
			defer cancel()

			return next(
				toolCtx,
				invocation,
			)
		}
	}
}
