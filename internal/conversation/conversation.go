package conversation

import "go-llm/internal/llm"

type Conversation struct {
	messages []llm.Message
}

func New() *Conversation {
	return &Conversation{
		messages: make([]llm.Message, 0),
	}
}

func (c *Conversation) AddMessage(
	msg llm.Message,
) {
	c.messages = append(c.messages, msg)
}

func (c *Conversation) AddUserMessage(
	content string,
) {
	c.AddMessage(
		llm.Message{
			Role:    llm.RoleUser,
			Content: content,
		},
	)
}

func (c *Conversation) AddAssistantMessage(
	content string,
) {
	c.AddMessage(
		llm.Message{
			Role:    llm.RoleAssistant,
			Content: content,
		},
	)
}

func (c *Conversation) AddToolMessage(
	toolName string,
	content string,
) {
	c.AddMessage(
		llm.Message{
			Role:     llm.RoleTool,
			ToolName: toolName,
			Content:  content,
		},
	)
}

func (c *Conversation) Messages() []llm.Message {
	return c.messages
}

func NewWithSystemPrompt(
	prompt string,
) *Conversation {

	c := New()

	c.AddSystemMessage(prompt)

	return c
}

func (c *Conversation) AddSystemMessage(
	content string,
) {
	c.AddMessage(
		llm.Message{
			Role:    llm.RoleSystem,
			Content: content,
		},
	)
}
