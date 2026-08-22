package conversation

import (
	"testing"

	"go-llm/internal/llm"
)

func TestConversationConstructors(t *testing.T) {
	tests := []struct {
		name          string
		systemPrompt  string
		expectedCount int
		expectedRole  llm.Role
		expectedText  string
	}{
		{
			name:          "empty conversation",
			systemPrompt:  "",
			expectedCount: 0,
		},
		{
			name:          "conversation with system prompt",
			systemPrompt:  "You are a helpful assistant.",
			expectedCount: 1,
			expectedRole:  llm.RoleSystem,
			expectedText:  "You are a helpful assistant.",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {

				var conv *Conversation

				if tt.systemPrompt == "" {
					conv = New()
				} else {
					conv = NewWithSystemPrompt(tt.systemPrompt)
				}

				if conv == nil {
					t.Fatal("conversation should not be nil")
				}

				if conv.MessageCount() != tt.expectedCount {
					t.Fatalf(
						"expected %d messages, got %d",
						tt.expectedCount,
						conv.MessageCount(),
					)
				}

				if tt.expectedCount == 0 {
					return
				}

				msg, ok := conv.LastMessage()

				if !ok {
					t.Fatal("expected last message")
				}

				if msg.Role != tt.expectedRole {
					t.Fatalf(
						"expected role %s got %s",
						tt.expectedRole,
						msg.Role,
					)
				}

				if msg.Content != tt.expectedText {
					t.Fatalf(
						"expected %q got %q",
						tt.expectedText,
						msg.Content,
					)
				}
			},
		)
	}
}

func TestAddMessages(t *testing.T) {

	tests := []struct {
		name    string
		add     func(*Conversation)
		role    llm.Role
		content string
	}{
		{
			name: "system",
			add: func(c *Conversation) {
				c.AddSystemMessage("system")
			},
			role:    llm.RoleSystem,
			content: "system",
		},
		{
			name: "user",
			add: func(c *Conversation) {
				c.AddUserMessage("hello")
			},
			role:    llm.RoleUser,
			content: "hello",
		},
		{
			name: "assistant",
			add: func(c *Conversation) {
				c.AddAssistantMessage("hi")
			},
			role:    llm.RoleAssistant,
			content: "hi",
		},
	}

	for _, tt := range tests {

		t.Run(
			tt.name, func(t *testing.T) {

				conv := New()

				tt.add(conv)

				if conv.MessageCount() != 1 {
					t.Fatalf(
						"expected 1 message got %d",
						conv.MessageCount(),
					)
				}

				msg, ok := conv.LastMessage()

				if !ok {
					t.Fatal("expected last message")
				}

				if msg.Role != tt.role {
					t.Fatalf(
						"expected role %s got %s",
						tt.role,
						msg.Role,
					)
				}

				if msg.Content != tt.content {
					t.Fatalf(
						"expected %q got %q",
						tt.content,
						msg.Content,
					)
				}
			},
		)
	}
}

func TestRemoveLastMessage(t *testing.T) {

	tests := []struct {
		name          string
		setup         func(*Conversation)
		expectedOK    bool
		expectedCount int
	}{
		{
			name:          "empty conversation",
			setup:         func(c *Conversation) {},
			expectedOK:    false,
			expectedCount: 0,
		},
		{
			name: "one message",
			setup: func(c *Conversation) {
				c.AddUserMessage("hello")
			},
			expectedOK:    true,
			expectedCount: 0,
		},
		{
			name: "two messages",
			setup: func(c *Conversation) {
				c.AddUserMessage("hello")
				c.AddAssistantMessage("hi")
			},
			expectedOK:    true,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {

		t.Run(
			tt.name, func(t *testing.T) {

				conv := New()

				tt.setup(conv)

				ok := conv.RemoveLastMessage()

				if ok != tt.expectedOK {
					t.Fatalf(
						"expected %v got %v",
						tt.expectedOK,
						ok,
					)
				}

				if conv.MessageCount() != tt.expectedCount {
					t.Fatalf(
						"expected %d messages got %d",
						tt.expectedCount,
						conv.MessageCount(),
					)
				}
			},
		)
	}
}

func TestClear(t *testing.T) {

	conv := New()

	conv.AddUserMessage("hello")
	conv.AddAssistantMessage("hi")

	conv.Clear()

	if !conv.IsEmpty() {
		t.Fatal("conversation should be empty")
	}

	if conv.MessageCount() != 0 {
		t.Fatal("expected zero messages")
	}
}

func TestMessagesReturnsCopy(t *testing.T) {

	conv := New()

	conv.AddUserMessage("hello")

	msgs := conv.Messages()

	msgs[0].Content = "modified"

	last, _ := conv.LastMessage()

	if last.Content != "hello" {
		t.Fatal("internal slice should not be modified")
	}
}
