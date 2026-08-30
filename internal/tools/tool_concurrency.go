package tools

import (
	"context"

	"go-llm/internal/llm"
)

func ToolConcurrency(
	maxConcurrency int,
) ToolMiddleware {

	if maxConcurrency <= 0 {
		return func(next Handler) Handler {
			return next
		}
	}

	slots := make(chan struct{}, maxConcurrency)

	return func(next Handler) Handler {

		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			select {
			case slots <- struct{}{}:
				defer func() {
					<-slots
				}()

			case <-ctx.Done():
				return nil, ctx.Err()
			}

			return next(
				ctx,
				invocation,
			)
		}
	}
}
