package tools

import (
	"context"
	"sync"
	"time"

	"go-llm/internal/llm"
)

type RateLimitPolicy struct {
	// Maximum number of tokens that can accumulate.
	Burst int

	// Tokens added per second.
	Rate time.Duration
}

type rateLimiter struct {
	mu sync.Mutex

	burst int

	tokens float64

	ratePerSecond float64

	last time.Time
}

func ToolRateLimit(
	policy RateLimitPolicy,
) ToolMiddleware {

	if policy.Burst <= 0 || policy.Rate <= 0 {
		return func(next Handler) Handler {
			return next
		}
	}

	limiter := &rateLimiter{
		burst:         policy.Burst,
		tokens:        float64(policy.Burst),
		ratePerSecond: float64(time.Second) / float64(policy.Rate),
		last:          time.Now(),
	}

	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			if err := limiter.wait(ctx); err != nil {
				return nil, err
			}

			return next(
				ctx,
				invocation,
			)
		}
	}
}

func (r *rateLimiter) wait(
	ctx context.Context,
) error {

	for {
		r.mu.Lock()

		now := time.Now()

		elapsed := now.Sub(r.last)
		r.last = now

		r.tokens += elapsed.Seconds() * r.ratePerSecond

		if r.tokens > float64(r.burst) {
			r.tokens = float64(r.burst)
		}

		if r.tokens >= 1 {
			r.tokens--

			r.mu.Unlock()

			return nil
		}

		needed := 1 - r.tokens

		wait := time.Duration(
			needed / r.ratePerSecond * float64(time.Second),
		)

		if wait <= 0 {
			wait = time.Nanosecond
		}

		r.mu.Unlock()

		timer := time.NewTimer(wait)

		select {
		case <-timer.C:

		case <-ctx.Done():

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			return ctx.Err()
		}
	}
}
