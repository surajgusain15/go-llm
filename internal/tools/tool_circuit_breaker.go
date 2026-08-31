package tools

import (
	"context"
	"errors"
	"sync"
	"time"

	"go-llm/internal/llm"
)

type CircuitBreakerPolicy struct {
	FailureThreshold int
	OpenTimeout      time.Duration
}

type circuitBreakerState uint8

const (
	circuitClosed circuitBreakerState = iota
	circuitOpen
	circuitHalfOpen
)

var ErrCircuitOpen = errors.New("tool circuit breaker is open")

type circuitBreaker struct {
	mu sync.Mutex

	state circuitBreakerState

	failures int

	openedAt time.Time

	probeInFlight bool

	policy CircuitBreakerPolicy
}

func ToolCircuitBreaker(
	policy CircuitBreakerPolicy,
) ToolMiddleware {

	if policy.FailureThreshold < 1 {
		policy.FailureThreshold = 1
	}

	if policy.OpenTimeout < 0 {
		policy.OpenTimeout = 0
	}

	cb := &circuitBreaker{
		policy: policy,
	}

	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			if err := cb.beforeCall(); err != nil {
				return nil, err
			}

			result, err := next(
				ctx,
				invocation,
			)

			cb.afterCall(err)

			return result, err
		}
	}
}

func (cb *circuitBreaker) beforeCall() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case circuitClosed:
		return nil

	case circuitOpen:
		if cb.policy.OpenTimeout <= 0 {
			cb.state = circuitHalfOpen
			cb.probeInFlight = true
			return nil
		}

		if time.Since(cb.openedAt) < cb.policy.OpenTimeout {
			return ErrCircuitOpen
		}

		cb.state = circuitHalfOpen

		if cb.probeInFlight {
			return ErrCircuitOpen
		}

		cb.probeInFlight = true

		return nil

	case circuitHalfOpen:
		if cb.probeInFlight {
			return ErrCircuitOpen
		}

		cb.probeInFlight = true

		return nil

	default:
		return ErrCircuitOpen
	}
}

func (cb *circuitBreaker) afterCall(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Caller cancellation is not evidence that the downstream
	// tool is unhealthy.
	if errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		if cb.state == circuitHalfOpen {
			cb.probeInFlight = false
		}

		return
	}

	switch cb.state {
	case circuitClosed:
		if err == nil {
			cb.failures = 0
			return
		}

		cb.failures++

		if cb.failures >= cb.policy.FailureThreshold {
			cb.state = circuitOpen
			cb.openedAt = time.Now()
		}

	case circuitHalfOpen:
		cb.probeInFlight = false

		if err == nil {
			cb.state = circuitClosed
			cb.failures = 0
			cb.openedAt = time.Time{}
			return
		}

		cb.state = circuitOpen
		cb.openedAt = time.Now()

	case circuitOpen:
		// Normally impossible because an open circuit does not
		// execute calls. Keep this defensive.
	}
}
