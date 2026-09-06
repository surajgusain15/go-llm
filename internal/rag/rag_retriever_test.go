package rag

import (
	"context"
	"errors"
	"testing"
)

func TestRAGRetriever_AppliesContextBudget(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	documents := []Document{
		{
			ID:      "doc-1",
			Content: "12345678", // 2 approximate tokens
			Vector:  []float32{1, 0},
		},
		{
			ID:      "doc-2",
			Content: "12345678", // 2 approximate tokens
			Vector:  []float32{1, 0},
		},
		{
			ID:      "doc-3",
			Content: "12345678", // 2 approximate tokens
			Vector:  []float32{1, 0},
		},
	}

	for _, document := range documents {
		if err := store.Add(document); err != nil {
			t.Fatal(err)
		}
	}

	retriever := NewRetriever(embedder, store)

	budget := NewContextBudget(
		ApproximateTokenCounter{},
		4,
	)

	ragRetriever := NewRAGRetriever(retriever, budget)

	got, err := ragRetriever.Retrieve(
		context.Background(),
		"database",
		RetrievalOptions{
			TopK: 3,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got.Chunks))
	}

	if got.Chunks[0].Document.ID != "doc-1" {
		t.Fatalf("expected doc-1, got %s", got.Chunks[0].Document.ID)
	}

	if got.Chunks[1].Document.ID != "doc-2" {
		t.Fatalf("expected doc-2, got %s", got.Chunks[1].Document.ID)
	}
}

type errorTestEmbedder struct {
	err error
}

func (e *errorTestEmbedder) Embed(
	ctx context.Context,
	text string,
) ([]float32, error) {
	return nil, e.err
}

func TestRAGRetriever_PropagatesRetrievalError(t *testing.T) {
	expectedErr := errors.New("embedding failed")

	embedder := &errorTestEmbedder{
		err: expectedErr,
	}

	store := NewInMemoryVectorStore()

	retriever := NewRetriever(embedder, store)

	budget := NewContextBudget(
		ApproximateTokenCounter{},
		100,
	)

	ragRetriever := NewRAGRetriever(retriever, budget)

	_, err := ragRetriever.Retrieve(
		context.Background(),
		"database",
		RetrievalOptions{
			TopK: 3,
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestRAGRetriever_ReturnsEmptyContextWhenNothingRetrieved(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	retriever := NewRetriever(embedder, store)

	budget := NewContextBudget(
		ApproximateTokenCounter{},
		100,
	)

	ragRetriever := NewRAGRetriever(retriever, budget)

	got, err := ragRetriever.Retrieve(
		context.Background(),
		"database",
		RetrievalOptions{
			TopK: 3,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(got.Chunks) != 0 {
		t.Fatalf("expected empty context, got %d chunks", len(got.Chunks))
	}

	if got.Text() != "" {
		t.Fatalf("expected empty context text, got %q", got.Text())
	}
}

func TestRAGRetriever_HonorsTopK(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	for i := 1; i <= 3; i++ {
		err := store.Add(
			Document{
				ID:      "doc-" + string(rune('0'+i)),
				Content: "document content",
				Vector:  []float32{1, 0},
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	retriever := NewRetriever(embedder, store)

	budget := NewContextBudget(
		ApproximateTokenCounter{},
		100,
	)

	ragRetriever := NewRAGRetriever(retriever, budget)

	got, err := ragRetriever.Retrieve(
		context.Background(),
		"database",
		RetrievalOptions{
			TopK: 2,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(got.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got.Chunks))
	}
}
