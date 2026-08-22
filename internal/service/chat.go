package service

import (
	"context"
	"strings"

	"go-llm/internal/conversation"
	"go-llm/internal/llm"
)

type ChatService struct {
	client llm.Client
}

func NewChatService(client llm.Client) *ChatService {
	return &ChatService{
		client: client,
	}
}

func (s *ChatService) Chat(
	ctx context.Context,
	conv *conversation.Conversation,
	message string,
) (string, error) {

	// Add the user's message to the conversation history.
	conv.AddUserMessage(message)

	// Send the entire conversation to the LLM.
	resp, err := s.client.Chat(
		ctx, llm.ChatRequest{
			Messages: conv.Messages(),
			Stream:   false,
		},
	)
	if err != nil {
		return "", err
	}

	// Save the assistant's response in the conversation.
	conv.AddAssistantMessage(resp.Message.Content)

	// Return the response to the caller.
	return resp.Message.Content, nil
}

func (s *ChatService) Stream(
	ctx context.Context,
	conv *conversation.Conversation,
	message string,
) <-chan llm.StreamResult {

	stream := make(chan llm.StreamResult)

	go func() {

		defer close(stream)

		conv.AddUserMessage(message)

		var builder strings.Builder

		clientStream := s.client.Stream(
			ctx,
			llm.ChatRequest{
				Messages: conv.Messages(),
				Stream:   true,
			},
		)

		for result := range clientStream {

			if result.Err != nil {
				stream <- result
				return
			}

			builder.WriteString(
				result.Chunk.Message.Content,
			)

			stream <- result

			if result.Chunk.Done {

				conv.AddAssistantMessage(
					builder.String(),
				)

				return
			}
		}
	}()

	return stream
}
