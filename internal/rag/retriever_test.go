package rag

import (
	"context"
	"errors"
	"testing"
)

func TestRetriever_EmbedsQueryAndReturnsTopK(
	t *testing.T,
) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database timeout": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	documents := []Document{
		{
			ID:      "timeout",
			Content: "database timeout is five seconds",
			Vector:  []float32{1, 0},
		},
		{
			ID:      "latency",
			Content: "database latency is low",
			Vector:  []float32{0.9, 0.1},
		},
		{
			ID:      "unrelated",
			Content: "UUID generation is supported",
			Vector:  []float32{0, 1},
		},
	}

	for _, document := range documents {
		if err := store.Add(document); err != nil {
			t.Fatal(err)
		}
	}

	retriever := NewRetriever(
		embedder,
		store,
	)

	results, err := retriever.Search(
		context.Background(),
		"database timeout",
		2,
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if len(results) != 2 {
		t.Fatalf(
			"expected 2 results, got %d",
			len(results),
		)
	}

	if results[0].Document.ID != "timeout" {
		t.Fatalf(
			"expected timeout first, got %q",
			results[0].Document.ID,
		)
	}

	if results[1].Document.ID != "latency" {
		t.Fatalf(
			"expected latency second, got %q",
			results[1].Document.ID,
		)
	}
}

func TestRetriever_ReturnsNilForEmptyQuery(
	t *testing.T,
) {
	retriever := NewRetriever(
		&testEmbedder{},
		NewInMemoryVectorStore(),
	)

	results, err := retriever.Search(
		context.Background(),
		"",
		5,
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if results != nil {
		t.Fatalf(
			"expected nil results, got %v",
			results,
		)
	}
}

func TestRetriever_ReturnsNilForInvalidTopK(
	t *testing.T,
) {
	retriever := NewRetriever(
		&testEmbedder{},
		NewInMemoryVectorStore(),
	)

	results, err := retriever.Search(
		context.Background(),
		"query",
		0,
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if results != nil {
		t.Fatalf(
			"expected nil results, got %v",
			results,
		)
	}
}

func TestRetriever_PropagatesEmbeddingError(
	t *testing.T,
) {
	expectedErr := errors.New(
		"embedding service unavailable",
	)

	retriever := NewRetriever(
		&failingEmbedder{
			err: expectedErr,
		},
		NewInMemoryVectorStore(),
	)

	results, err := retriever.Search(
		context.Background(),
		"database timeout",
		5,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected embedding error, got %v",
			err,
		)
	}

	if results != nil {
		t.Fatalf(
			"expected nil results, got %v",
			results,
		)
	}
}
