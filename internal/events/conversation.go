package events

type UserMessage struct {
	BaseEvent

	Content string
}

func NewUserMessage(
	content string,
) UserMessage {

	return UserMessage{
		BaseEvent: NewBaseEvent(),
		Content:   content,
	}
}

func (UserMessage) Type() EventType {
	return "conversation.user"
}

type AssistantMessage struct {
	BaseEvent

	Content string
}

func NewAssistantMessage(
	content string,
) AssistantMessage {

	return AssistantMessage{
		BaseEvent: NewBaseEvent(),
		Content:   content,
	}
}

func (AssistantMessage) Type() EventType {
	return "conversation.assistant"
}
