package events

import "time"

type EventType string

const (
	EventLLMRequestStarted  EventType = "llm.request.started"
	EventLLMRequestFinished EventType = "llm.request.finished"

	EventToolStarted  EventType = "tool.started"
	EventToolFinished EventType = "tool.finished"

	EventConversationUser      EventType = "conversation.user"
	EventConversationAssistant EventType = "conversation.assistant"
)

type Event interface {
	Type() EventType
	Timestamp() time.Time
}
