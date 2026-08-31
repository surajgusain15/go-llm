package tools

func WithToolCircuitBreaker(
	policy CircuitBreakerPolicy,
) ExecutorOption {

	return func(e *Executor) {
		e.middlewares = append(
			e.middlewares,
			ToolCircuitBreaker(policy),
		)
	}
}

func WithToolRateLimit(
	policy RateLimitPolicy,
) ExecutorOption {
	return func(e *Executor) {
		e.middlewares = append(
			e.middlewares,
			ToolRateLimit(policy),
		)
	}
}

func WithToolBulkhead(
	policy ToolBulkheadPolicy,
) ExecutorOption {

	return func(e *Executor) {
		e.middlewares = append(
			e.middlewares,
			ToolBulkhead(policy),
		)
	}
}
