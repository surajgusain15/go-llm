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
