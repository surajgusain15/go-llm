package service

import (
	"context"
	"strings"

	"go-llm/internal/conversation"
	"go-llm/internal/llm"
)

const DefaultSystemPrompt = `
You are a helpful AI assistant.
`

const maxToolIterations = 10

type ChatService struct {
	client   llm.Client
	executor ToolExecutor
}

func NewChatService(
	client llm.Client,
	executor ToolExecutor,
) *ChatService {

	return &ChatService{
		client:   client,
		executor: executor,
	}
}

func (s *ChatService) Chat(
	ctx context.Context,
	conv *conversation.Conversation,
	message string,
) (string, error) {

	conv.AddUserMessage(message)

	return s.executeAgentLoop(
		ctx,
		conv,
	)
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

func (s *ChatService) NewConversation() *conversation.Conversation {

	return conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)
}
