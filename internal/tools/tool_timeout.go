package tools

import (
	"context"
	"fmt"
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

			timeoutFired := make(chan struct{})

			timer := time.AfterFunc(
				timeout,
				func() {
					close(timeoutFired)
				},
			)

			defer timer.Stop()

			result, err := next(
				toolCtx,
				invocation,
			)

			if err == nil {
				return result, nil
			}

			// The child context can be cancelled either by:
			//
			//   1. our timeout
			//   2. parent cancellation/deadline
			//
			// timeoutFired distinguishes the two cases.
			select {
			case <-timeoutFired:
				return nil, fmt.Errorf(
					"%w: %w",
					ErrToolTimeout,
					err,
				)

			default:
				return result, err
			}
		}
	}
}
