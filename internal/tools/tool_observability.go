package tools

import (
	"context"
	"time"

	"go-llm/internal/events"
	"go-llm/internal/llm"
)

func ToolObservability() ToolMiddleware {
	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			middlewareCtx := toolMiddlewareContext(ctx)

			if middlewareCtx.Core != nil {
				middlewareCtx.Core.Emit(
					events.NewToolStarted(
						invocation.Name,
					),
				)
			}

			start := time.Now()

			result, err := next(
				ctx,
				invocation,
			)

			duration := time.Since(start)

			if middlewareCtx.Core != nil {
				middlewareCtx.Core.Emit(
					events.NewToolFinished(
						invocation.Name,
						duration,
						err,
					),
				)
			}

			return result, err
		}
	}
}
