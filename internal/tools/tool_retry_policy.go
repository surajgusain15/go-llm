package tools

import (
	"context"
	"errors"
	"time"

	"go-llm/internal/events"
	"go-llm/internal/llm"
)

type ToolRetryPolicy struct {
	MaxAttempts int

	ShouldRetry func(error) bool

	Backoff BackoffPolicy

	Budget *ToolRetryBudgetPolicy
}

func ToolRetry(
	policy ToolRetryPolicy,
) ToolMiddleware {

	var budget *toolRetryBudget

	if policy.Budget != nil {
		budget = newToolRetryBudget(
			*policy.Budget,
		)
	}

	return func(next Handler) Handler {

		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			maxAttempts := max(policy.MaxAttempts, 1)

			for attempt := 1; attempt <= maxAttempts; attempt++ {

				result, err := next(
					ctx,
					invocation,
				)

				if err == nil {
					return result, nil
				}

				if errors.Is(err, context.Canceled) ||
					errors.Is(err, context.DeadlineExceeded) {
					return nil, err
				}

				if attempt == maxAttempts {
					return nil, err
				}

				if policy.ShouldRetry == nil ||
					!policy.ShouldRetry(err) {
					return nil, err
				}

				var releaseBudget func()

				if budget != nil {
					var acquireErr error

					releaseBudget, acquireErr = budget.acquire(
						ctx,
						invocation.Name,
					)

					if acquireErr != nil {
						return nil, acquireErr
					}
				}

				delay := time.Duration(0)

				if policy.Backoff != nil {
					delay = policy.Backoff(attempt)
				}

				middlewareCtx := toolMiddlewareContext(ctx)

				if middlewareCtx.Core != nil {
					middlewareCtx.Core.Emit(
						events.NewToolRetry(
							invocation.Name,
							attempt,
							delay,
							err,
						),
					)
				}

				if delay > 0 {
					timer := time.NewTimer(delay)

					select {
					case <-timer.C:

					case <-ctx.Done():
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}

						if releaseBudget != nil {
							releaseBudget()
						}

						return nil, ctx.Err()
					}
				}

				if releaseBudget != nil {
					releaseBudget()
				}
			}

			return nil, context.Canceled
		}
	}
}

func WithToolRetry(
	policy ToolRetryPolicy,
) ExecutorOption {

	return func(e *Executor) {
		e.middlewares = append(
			e.middlewares,
			ToolRetry(policy),
		)
	}
}
