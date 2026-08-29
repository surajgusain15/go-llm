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

	EventAgentIterationStarted EventType = "agent.iteration.started"
	EventAgentToolsPlanned     EventType = "agent.tools.planned"

	EventAgentStarted    EventType = "agent.started"
	EventAgentFinished   EventType = "agent.finished"
	EventThinkingStarted EventType = "thinking.started"

	EventDatabaseQueryStarted  EventType = "database.query.started"
	EventDatabaseQueryFinished EventType = "database.query.finished"
)

type Event interface {
	Type() EventType
	Timestamp() time.Time
}
