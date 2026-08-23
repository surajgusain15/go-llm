package llm

import "fmt"

type ErrorCode string

const (
	ErrorInvalidRequest   ErrorCode = "invalid_request"
	ErrorAuthentication   ErrorCode = "authentication"
	ErrorRateLimited      ErrorCode = "rate_limited"
	ErrorModelUnsupported ErrorCode = "model_unsupported"
	ErrorInternal         ErrorCode = "internal"
)

type APIError struct {
	Code       ErrorCode
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf(
		"LLM API error (%d): %s",
		e.StatusCode,
		e.Message,
	)
}
