package rag

import "testing"

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

	want := "Database connections should be closed.\n\nConnection pooling improves performance."

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

	want := "Highest ranked content.\n\nSecond ranked content."

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
