package service

import (
	"context"
	"errors"
	"time"

	"go-llm/internal/conversation"
	"go-llm/internal/events"
)

const maxAgentIterations = 10

func (s *ChatService) executeAgentLoop(
	ctx context.Context,
	conv *conversation.Conversation,
) (string, error) {

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
		"maximum tool iterations exceeded",
	)
}
