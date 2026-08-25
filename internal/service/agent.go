package service

import (
	"context"
	"errors"
	"time"

	"go-llm/internal/conversation"
)

const maxAgentIterations = 10

func (s *ChatService) executeAgentLoop(
	ctx context.Context,
	conv *conversation.Conversation,
) (string, error) {

	for range maxAgentIterations {

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
