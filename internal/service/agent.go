package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-llm/internal/conversation"
	"go-llm/internal/events"
	"go-llm/internal/llm"
)

const maxAgentIterations = 10

func (s *ChatService) executeAgentLoop(
	ctx context.Context,
	conv *conversation.Conversation,
) (string, error) {

	fmt.Print("Schemas", s.executor.Schemas())

	for range maxAgentIterations {

		req := llm.ChatRequest{
			Messages: conv.Messages(),
			Tools:    s.executor.Schemas(),
			Stream:   false,
		}

		start := time.Now()

		s.core.Observer.OnEvent(
			events.NewLLMRequestStarted(
				req.Model,
			),
		)

		resp, err := s.client.Chat(ctx, req)

		events.NewLLMRequestFinished(
			req.Model,
			time.Since(start),
			err,
		)

		if err != nil {
			return "", err
		}

		conv.AddMessage(resp.Message)

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
