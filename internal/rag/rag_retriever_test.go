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

	if got.RetrievedCount != 3 {
		t.Fatalf("expected 3 retrieved results, got %d", got.RetrievedCount)
	}

	if got.SelectedCount != 2 {
		t.Fatalf("expected 2 selected results, got %d", got.SelectedCount)
	}

	if len(got.Context.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got.Context.Chunks))
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

	if got.RetrievedCount != 0 {
		t.Fatalf(
			"expected 0 retrieved results, got %d",
			got.RetrievedCount,
		)
	}

	if got.SelectedCount != 0 {
		t.Fatalf(
			"expected 0 selected results, got %d",
			got.SelectedCount,
		)
	}
	if len(got.Context.Chunks) != 0 {
		t.Fatalf(
			"expected empty context, got %d chunks",
			len(got.Context.Chunks),
		)
	}

	if got.Context.Text() != "" {
		t.Fatalf(
			"expected empty context text, got %q",
			got.Context.Text(),
		)
	}
}

func TestRAGRetriever_DistinguishesRetrievedFromSelected(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	err := store.Add(
		Document{
			ID:      "doc-1",
			Content: "12345678901234567890", // 5 approximate tokens
			Vector:  []float32{1, 0},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	retriever := NewRetriever(embedder, store)

	// Budget is smaller than the first result.
	budget := NewContextBudget(
		ApproximateTokenCounter{},
		4,
	)

	ragRetriever := NewRAGRetriever(retriever, budget)

	got, err := ragRetriever.Retrieve(
		context.Background(),
		"database",
		RetrievalOptions{
			TopK: 1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got.RetrievedCount != 1 {
		t.Fatalf(
			"expected 1 retrieved result, got %d",
			got.RetrievedCount,
		)
	}

	if got.SelectedCount != 0 {
		t.Fatalf(
			"expected 0 selected results, got %d",
			got.SelectedCount,
		)
	}

	if len(got.Context.Chunks) != 0 {
		t.Fatalf(
			"expected empty context, got %d chunks",
			len(got.Context.Chunks),
		)
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
			TopK: 3,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.RetrievedCount != 3 {
		t.Fatalf(
			"expected 3 retrieved results, got %d",
			got.RetrievedCount,
		)
	}

	if got.SelectedCount != 3 {
		t.Fatalf(
			"expected 3 selected results, got %d",
			got.SelectedCount,
		)
	}

	if len(got.Context.Chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(got.Context.Chunks))
	}
}

func TestRAGRetriever_ReturnsNoMatchWhenNothingRetrieved(t *testing.T) {
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

	if got.RetrievedCount != 0 {
		t.Fatalf(
			"expected 0 retrieved results, got %d",
			got.RetrievedCount,
		)
	}

	if got.SelectedCount != 0 {
		t.Fatalf(
			"expected 0 selected results, got %d",
			got.SelectedCount,
		)
	}

	if len(got.Context.Chunks) != 0 {
		t.Fatalf(
			"expected empty context, got %d chunks",
			len(got.Context.Chunks),
		)
	}

	if got.Context.Text() != "" {
		t.Fatalf(
			"expected empty context text, got %q",
			got.Context.Text(),
		)
	}
}
