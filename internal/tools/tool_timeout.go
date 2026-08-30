package tools

import (
	"context"
	"maps"
	"time"

	"go-llm/internal/llm"
)

func ToolTimeouts(
	defaultTimeout time.Duration,
	timeouts map[string]time.Duration,
) ToolMiddleware {

	configuredTimeouts := make(
		map[string]time.Duration,
		len(timeouts),
	)

	maps.Copy(configuredTimeouts, timeouts)

	return func(next Handler) Handler {

		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			timeout := defaultTimeout

			if configured, ok := configuredTimeouts[invocation.Name]; ok {
				timeout = configured
			}

			if timeout <= 0 {
				return next(ctx, invocation)
			}

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
