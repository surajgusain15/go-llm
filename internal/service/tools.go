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

	for _, call := range calls {

		result, err := s.executeToolCall(
			ctx,
			call,
		)
		if err != nil {
			return err
		}

		err = s.appendToolResult(
			conv,
			call,
			result,
		)
		if err != nil {
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
) error {

	content, err := toolResultToString(
		result.Content,
	)
	if err != nil {
		return err
	}

	conv.AddToolMessage(
		call.Function.Name,
		content,
	)

	return nil
}
