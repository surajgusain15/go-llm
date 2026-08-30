package tools

import (
	"context"
	"time"

	"go-llm/internal/llm"
)

func ToolTimeouts(
	defaultTimeout time.Duration,
	timeouts map[string]time.Duration,
) ToolMiddleware {

	configured := make(map[string]time.Duration, len(timeouts))

	for name, timeout := range timeouts {
		configured[name] = timeout
	}

	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			timeout := defaultTimeout

			if configuredTimeout, ok := configured[invocation.Name]; ok {
				timeout = configuredTimeout
			}

			if timeout <= 0 {
				return next(ctx, invocation)
			}

			toolCtx, cancel := context.WithTimeout(
				ctx,
				timeout,
			)
			defer cancel()

			return next(toolCtx, invocation)
		}
	}
}
