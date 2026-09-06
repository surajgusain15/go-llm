package rag

import (
	"testing"
)

func TestContextBudget_SelectsResultsWithinLimit(t *testing.T) {
	counter := ApproximateTokenCounter{}
	budget := NewContextBudget(counter, 10)

	results := []SearchResult{
		{
			Document: Document{
				ID:      "doc-1",
				Content: "12345678", // 2 tokens
			},
		},
		{
			Document: Document{
				ID:      "doc-2",
				Content: "123456789012", // 3 tokens
			},
		},
		{
			Document: Document{
				ID:      "doc-3",
				Content: "1234567890123456", // 4 tokens
			},
		},
	}

	got := budget.Select(results)

	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}
}

func TestContextBudget_DoesNotExceedLimit(t *testing.T) {
	counter := ApproximateTokenCounter{}
	budget := NewContextBudget(counter, 5)

	results := []SearchResult{
		{
			Document: Document{
				ID:      "doc-1",
				Content: "12345678", // 2
			},
		},
		{
			Document: Document{
				ID:      "doc-2",
				Content: "123456789012", // 3
			},
		},
		{
			Document: Document{
				ID:      "doc-3",
				Content: "1234567890123456", // 4
			},
		},
	}

	got := budget.Select(results)

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	if got[0].Document.ID != "doc-1" {
		t.Fatalf("expected doc-1, got %s", got[0].Document.ID)
	}

	if got[1].Document.ID != "doc-2" {
		t.Fatalf("expected doc-2, got %s", got[1].Document.ID)
	}
}

func TestContextBudget_PreservesRankingOrder(t *testing.T) {
	counter := ApproximateTokenCounter{}
	budget := NewContextBudget(counter, 5)

	results := []SearchResult{
		{
			Document: Document{
				ID:      "highest-ranked",
				Content: "12345678",
			},
			Similarity: 0.95,
		},
		{
			Document: Document{
				ID:      "second-ranked",
				Content: "12345678",
			},
			Similarity: 0.90,
		},
	}

	got := budget.Select(results)

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	if got[0].Document.ID != "highest-ranked" {
		t.Fatalf("ranking changed: got %s first", got[0].Document.ID)
	}

	if got[1].Document.ID != "second-ranked" {
		t.Fatalf("ranking changed: got %s second", got[1].Document.ID)
	}
}

func TestContextBudget_SkipsNothingAfterFirstOverflow(t *testing.T) {
	counter := ApproximateTokenCounter{}
	budget := NewContextBudget(counter, 5)

	results := []SearchResult{
		{
			Document: Document{
				ID:      "doc-1",
				Content: "1234567890123456", // 4
			},
		},
		{
			Document: Document{
				ID:      "doc-2",
				Content: "123456789012", // 3
			},
		},
		{
			Document: Document{
				ID:      "doc-3",
				Content: "1234", // 1
			},
		},
	}

	got := budget.Select(results)

	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}

	if got[0].Document.ID != "doc-1" {
		t.Fatalf("expected doc-1, got %s", got[0].Document.ID)
	}
}

func TestContextBudget_ReturnsNilForInvalidLimit(t *testing.T) {
	counter := ApproximateTokenCounter{}
	budget := NewContextBudget(counter, 0)

	results := []SearchResult{
		{
			Document: Document{
				ID:      "doc-1",
				Content: "hello",
			},
		},
	}

	got := budget.Select(results)

	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}
