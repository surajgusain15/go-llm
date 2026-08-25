package service

import (
	"context"

	"go-llm/internal/conversation"
	"go-llm/internal/core"
	"go-llm/internal/events"
	"go-llm/internal/llm"
)

const DefaultSystemPrompt = `
You are a helpful AI assistant.
`

type ChatService struct {
	client   llm.Client
	executor ToolExecutor
	core     *core.Core
}

func NewChatService(
	client llm.Client,
	executor ToolExecutor,
	rt *core.Core,
) *ChatService {

	if rt == nil {
		rt = core.New(events.NopObserver{})
	}

	return &ChatService{
		client:   client,
		executor: executor,
		core:     rt,
	}
}

func (s *ChatService) Chat(
	ctx context.Context,
	conv *conversation.Conversation,
	message string,
) (string, error) {

	conv.AddUserMessage(message)

	s.core.Emit(
		events.NewUserMessage(
			message,
		),
	)

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

	conv.AddUserMessage(message)

	s.core.Emit(
		events.NewUserMessage(message),
	)

	s.core.Emit(
		events.NewAgentStarted(),
	)

	return s.executeStreamingAgentLoop(
		ctx,
		conv,
	)
}

func (s *ChatService) NewConversation() *conversation.Conversation {

	return conversation.NewWithSystemPrompt(
		DefaultSystemPrompt,
	)
}

type ResponseWriter interface {
	Write(chunk llm.StreamChunk) error
	Close() error
}
