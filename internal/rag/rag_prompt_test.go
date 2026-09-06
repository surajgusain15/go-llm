package rag

import (
	"strings"
	"testing"
)

func TestRAGPrompt_IncludesContext(t *testing.T) {
	context := NewContext(
		[]SearchResult{
			{
				Document: Document{
					ID:      "doc-1",
					Content: "Database connections should be closed after every request.",
				},
			},
		},
	)

	prompt := NewRAGPrompt(
		"How should database connections be handled?",
		context,
	)

	text := prompt.Text()

	if !strings.Contains(text, "Database connections should be closed after every request.") {
		t.Fatalf("expected prompt to contain context, got %q", text)
	}
}

func TestRAGPrompt_IncludesQuery(t *testing.T) {
	query := "How should database connections be handled?"

	prompt := NewRAGPrompt(query, Context{})

	text := prompt.Text()

	if !strings.Contains(text, query) {
		t.Fatalf("expected prompt to contain query, got %q", text)
	}
}

func TestRAGPrompt_ContainsGroundingInstruction(t *testing.T) {
	prompt := NewRAGPrompt("What is a database timeout?", Context{})

	text := prompt.Text()

	expected := "Answer the question using only the provided context."

	if !strings.Contains(text, expected) {
		t.Fatalf("expected grounding instruction %q, got %q", expected, text)
	}
}

func TestRAGPrompt_HandlesEmptyContext(t *testing.T) {
	query := "What is a database timeout?"

	prompt := NewRAGPrompt(query, Context{})

	text := prompt.Text()

	if !strings.Contains(text, "Context:") {
		t.Fatalf("expected prompt to contain context section, got %q", text)
	}

	if !strings.Contains(text, query) {
		t.Fatalf("expected prompt to contain query, got %q", text)
	}
}

func TestRAGPrompt_ContainsInsufficientContextInstruction(t *testing.T) {
	prompt := NewRAGPrompt("What is a database timeout?", Context{})

	text := prompt.Text()

	expected := "If the answer cannot be found in the context, say that the information is not available in the provided context."

	if !strings.Contains(text, expected) {
		t.Fatalf(
			"expected insufficient-context instruction %q, got %q",
			expected,
			text,
		)
	}
}

func TestRAGPrompt_ContainsNoOutsideKnowledgeInstruction(t *testing.T) {
	prompt := NewRAGPrompt("What is a database timeout?", Context{})

	text := prompt.Text()

	expected := "Do not use outside knowledge or make up information."

	if !strings.Contains(text, expected) {
		t.Fatalf(
			"expected no-outside-knowledge instruction %q, got %q",
			expected,
			text,
		)
	}
}
