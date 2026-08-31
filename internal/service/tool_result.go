package service

import (
	"time"

	"go-llm/internal/llm"
)

type ToolExecutionResult struct {
	Call     llm.ToolCall
	Result   *llm.ToolResult
	Err      error
	Duration time.Duration
	Attempts int
}
