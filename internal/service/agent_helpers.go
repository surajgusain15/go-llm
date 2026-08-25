package service

import (
	"time"

	"go-llm/internal/conversation"
	"go-llm/internal/events"
	"go-llm/internal/llm"
)

func (s *ChatService) buildChatRequest(
	conv *conversation.Conversation,
	stream bool,
) llm.ChatRequest {

	return llm.ChatRequest{
		Messages: conv.Messages(),
		Tools:    s.executor.Schemas(),
		Stream:   stream,
	}
}

func (s *ChatService) emitLLMRequestStarted() {

	s.core.Emit(
		events.NewLLMRequestStarted(),
	)
}

func (s *ChatService) emitLLMRequestFinished(
	start time.Time,
	err error,
) {

	s.core.Emit(
		events.NewLLMRequestFinished(
			time.Since(start),
			err,
		),
	)
}
