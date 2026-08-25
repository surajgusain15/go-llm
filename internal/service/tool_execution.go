package service

import (
	"context"

	"go-llm/internal/llm"
)

func (s *ChatService) executeToolCall(
	ctx context.Context,
	call llm.ToolCall,
) (*llm.ToolResult, error) {

	result, err := s.executor.Execute(
		ctx,
		call.Function.Name,
		call.Function.Arguments,
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}
