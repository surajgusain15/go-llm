package service

import (
	"context"
	"errors"
	"fmt"

	"go-llm/internal/conversation"
	"go-llm/internal/llm"
)

const maxAgentIterations = 10

func (s *ChatService) executeAgentLoop(
	ctx context.Context,
	conv *conversation.Conversation,
) (string, error) {

	fmt.Print("Schemas", s.executor.Schemas())

	for range maxAgentIterations {

		resp, err := s.client.Chat(
			ctx,
			llm.ChatRequest{
				Messages: conv.Messages(),
				Tools:    s.executor.Schemas(),
				Stream:   false,
			},
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
