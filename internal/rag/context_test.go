package rag

import (
	"strings"
	"testing"
)

func TestContext_TextCombinesChunks(t *testing.T) {
	context := NewContext(
		[]SearchResult{
			{
				Document: Document{
					ID:      "doc-1",
					Content: "Database connections should be closed.",
				},
			},
			{
				Document: Document{
					ID:      "doc-2",
					Content: "Connection pooling improves performance.",
				},
			},
		},
	)

	got := context.Text()

	want := "Source: doc-1\nDatabase connections should be closed.\n\nSource: doc-2\nConnection pooling improves performance."
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestContext_TextPreservesRetrievalOrder(t *testing.T) {
	context := NewContext(
		[]SearchResult{
			{
				Document: Document{
					ID:      "highest",
					Content: "Highest ranked content.",
				},
			},
			{
				Document: Document{
					ID:      "second",
					Content: "Second ranked content.",
				},
			},
		},
	)

	got := context.Text()

	want := "Source: highest\nHighest ranked content.\n\nSource: second\nSecond ranked content."

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestContext_TextEmpty(t *testing.T) {
	context := NewContext(nil)

	if got := context.Text(); got != "" {
		t.Fatalf("expected empty context, got %q", got)
	}
}

func TestContext_TextIncludesSourceDocumentID(t *testing.T) {
	context := NewContext(
		[]SearchResult{
			{
				Document: Document{
					ID:      "doc-1#chunk-0",
					Content: "First chunk.",
				},
			},
			{
				Document: Document{
					ID:      "doc-1#chunk-1",
					Content: "Second chunk.",
				},
			},
		},
	)

	text := context.Text()

	if !strings.Contains(text, "Source: doc-1#chunk-0") {
		t.Fatalf("expected first source ID, got %q", text)
	}

	if !strings.Contains(text, "Source: doc-1#chunk-1") {
		t.Fatalf("expected second source ID, got %q", text)
	}
}
