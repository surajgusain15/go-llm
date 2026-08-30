package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go-llm/internal/conversation"
	"go-llm/internal/llm"
)

func (s *ChatService) executeToolCalls(
	ctx context.Context,
	conv *conversation.Conversation,
	calls []llm.ToolCall,
) error {

	results, err := s.executeTools(
		ctx,
		calls,
	)
	if err != nil {
		return err
	}

	for _, executed := range results {
		if err := s.appendToolResult(
			conv,
			executed.Call,
			executed.Result,
			executed.Err,
		); err != nil {
			return err
		}
	}

	return nil
}

func toolResultToString(
	value any,
) (string, error) {

	switch v := value.(type) {

	case string:
		return v, nil

	case []byte:
		return string(v), nil

	default:

		data, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf(
				"marshal tool result: %w",
				err,
			)
		}

		return string(data), nil
	}
}

func (s *ChatService) appendToolResult(
	conv *conversation.Conversation,
	call llm.ToolCall,
	result *llm.ToolResult,
	toolErr error,
) error {

	if toolErr != nil {
		conv.AddToolMessage(
			call.Function.Name,
			fmt.Sprintf(
				"Tool execution failed: %v",
				toolErr,
			),
		)

		return nil
	}

	content, err := toolResultToString(result.Content)
	if err != nil {
		return err
	}

	content = limitToolOutput(
		content,
		s.toolOutputLimit,
	)

	conv.AddToolMessage(
		call.Function.Name,
		content,
	)

	return nil
}
