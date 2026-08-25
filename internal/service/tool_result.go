package service

import "go-llm/internal/llm"

type ToolExecutionResult struct {
	Call   llm.ToolCall
	Result *llm.ToolResult
	Err    error
}
