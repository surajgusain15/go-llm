package tools

import (
	"context"
	"sync"
	"time"

	"go-llm/internal/llm"
)

type RateLimitPolicy struct {
	// Maximum number of tokens that can accumulate
	// for the global limiter.
	Burst int

	// Interval between tokens for the global limiter.
	Rate time.Duration

	// Optional per-tool overrides.
	//
	// If a tool is present here, its limiter is used
	// instead of the global limiter.
	PerTool map[string]RateLimitConfig
}

type RateLimitConfig struct {
	// Maximum number of tokens that can accumulate.
	Burst int

	// Interval between tokens.
	Rate time.Duration
}

type rateLimiter struct {
	mu sync.Mutex

	burst int

	tokens float64

	ratePerSecond float64

	last time.Time
}

func newRateLimiter(
	burst int,
	rate time.Duration,
) *rateLimiter {

	if burst <= 0 || rate <= 0 {
		return nil
	}

	return &rateLimiter{
		burst:         burst,
		tokens:        float64(burst),
		ratePerSecond: float64(time.Second) / float64(rate),
		last:          time.Now(),
	}
}

func ToolRateLimit(
	policy RateLimitPolicy,
) ToolMiddleware {

	globalLimiter := newRateLimiter(
		policy.Burst,
		policy.Rate,
	)

	perToolLimiters := make(
		map[string]*rateLimiter,
		len(policy.PerTool),
	)

	for name, config := range policy.PerTool {
		limiter := newRateLimiter(
			config.Burst,
			config.Rate,
		)

		if limiter != nil {
			perToolLimiters[name] = limiter
		}
	}

	return func(next Handler) Handler {

		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			limiter := globalLimiter

			if toolLimiter, ok := perToolLimiters[invocation.Name]; ok {
				limiter = toolLimiter
			}

			if limiter == nil {
				return next(
					ctx,
					invocation,
				)
			}

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
			needed / r.ratePerSecond *
				float64(time.Second),
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
