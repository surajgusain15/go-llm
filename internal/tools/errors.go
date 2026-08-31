package tools

import "errors"

var (
	ErrToolNotInteractive = errors.New("tool is not interactive")

	ErrToolNotFound = errors.New("tool not found")

	ErrToolInvalidInput = errors.New("invalid tool input")

	ErrToolTimeout = errors.New("tool timeout")

	ErrToolRateLimited = errors.New("tool rate limited")

	ErrToolConcurrencyLimit = errors.New(
		"tool concurrency limit exceeded",
	)

	ErrToolCircuitOpen = errors.New(
		"tool circuit breaker open",
	)

	ErrToolUnavailable = errors.New("tool unavailable")
	ErrToolTemporary   = errors.New("temporary tool failure")
)
