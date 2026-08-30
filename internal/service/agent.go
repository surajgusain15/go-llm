package service

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"go-llm/internal/conversation"
	"go-llm/internal/events"
)

const maxAgentIterations = 10
const maxToolCalls = 10

const DefaultToolOutputLimit = 64 * 1024
const DefaultToolConcurrency = 8

func limitToolOutput(
	content string,
	maxBytes int,
) string {

	if maxBytes <= 0 || len(content) <= maxBytes {
		return content
	}

	marker := "\n\n[tool output truncated: exceeded maximum size]"

	if maxBytes <= len(marker) {
		return marker[:maxBytes]
	}

	limit := maxBytes - len(marker)

	for limit > 0 && !utf8.ValidString(
		content[:limit],
	) {
		limit--
	}

	return content[:limit] + marker
}

func (s *ChatService) executeAgentLoop(
	ctx context.Context,
	conv *conversation.Conversation,
) (string, error) {

	toolCalls := 0

	for i := range maxAgentIterations {

		s.core.Emit(
			events.NewAgentIterationStarted(
				i + 1,
			),
		)

		req := s.buildChatRequest(
			conv,
			false,
		)

		start := time.Now()

		s.emitLLMRequestStarted()

		resp, err := s.client.Chat(
			ctx,
			req,
		)

		s.emitLLMRequestFinished(
			start,
			err,
		)

		if err != nil {
			return "", err
		}

		conv.AddMessage(
			resp.Message,
		)

		if len(resp.Message.ToolCalls) == 0 {

			s.core.Emit(
				events.NewAgentFinished(),
			)

			return resp.Message.Content, nil
		}

		if toolCalls+len(resp.Message.ToolCalls) > maxToolCalls {
			return "", errors.New(
				"maximum tool calls exceeded",
			)
		}

		toolCalls += len(resp.Message.ToolCalls)

		err = s.executeToolCalls(
			ctx,
			conv,
			resp.Message.ToolCalls,
		)
		if err != nil {
			return "", err
		}
	}

	return "", errors.New(
		"maximum agent iterations exceeded",
	)
}
