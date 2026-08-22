package conversation

import "go-llm/internal/llm"

type Conversation struct {
	messages []llm.Message
}

func New() *Conversation {
	return NewWithSystemPrompt("")
}

func NewWithSystemPrompt(prompt string) *Conversation {
	c := &Conversation{
		messages: make([]llm.Message, 0, 10),
	}

	if prompt != "" {
		c.AddSystemMessage(prompt)
	}

	return c
}

func (c *Conversation) AddSystemMessage(content string) {
	c.messages = append(
		c.messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: content,
		},
	)
}

func (c *Conversation) AddUserMessage(content string) {
	c.messages = append(
		c.messages, llm.Message{
			Role:    llm.RoleUser,
			Content: content,
		},
	)
}

func (c *Conversation) AddAssistantMessage(content string) {
	c.messages = append(
		c.messages, llm.Message{
			Role:    llm.RoleAssistant,
			Content: content,
		},
	)
}

// Messages returns a copy of the conversation messages
// to prevent callers from modifying internal state.
func (c *Conversation) Messages() []llm.Message {
	messages := make([]llm.Message, len(c.messages))
	copy(messages, c.messages)

	return messages
}

func (c *Conversation) LastMessage() (llm.Message, bool) {
	if len(c.messages) == 0 {
		return llm.Message{}, false
	}

	return c.messages[len(c.messages)-1], true
}

func (c *Conversation) RemoveLastMessage() bool {
	if len(c.messages) == 0 {
		return false
	}

	c.messages = c.messages[:len(c.messages)-1]

	return true
}

func (c *Conversation) Clear() {
	c.messages = c.messages[:0]
}

func (c *Conversation) MessageCount() int {
	return len(c.messages)
}

func (c *Conversation) IsEmpty() bool {
	return len(c.messages) == 0
}
