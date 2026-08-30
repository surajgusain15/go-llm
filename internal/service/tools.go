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
			executed,
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
	executed ToolExecutionResult,
) error {

	if executed.Err != nil {
		content := fmt.Sprintf(
			"Tool execution failed: %v",
			executed.Err,
		)

		conv.AddToolMessage(
			executed.Call.Function.Name,
			content,
		)

		return nil
	}

	if executed.Result == nil {
		return fmt.Errorf(
			"tool %q returned nil result without error",
			executed.Call.Function.Name,
		)
	}

	content, err := toolResultToString(
		executed.Result.Content,
	)
	if err != nil {
		return err
	}

	conv.AddToolMessage(
		executed.Call.Function.Name,
		content,
	)

	return nil
}
