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

	results, err := retriever.Retrieve(
		context.Background(),
		"database timeout",
		RetrievalOptions{
			TopK: 2,
		},
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

func TestRetriever_ReturnsErrorForEmptyQuery(
	t *testing.T,
) {
	retriever := NewRetriever(
		&testEmbedder{},
		NewInMemoryVectorStore(),
	)

	results, err := retriever.Retrieve(
		context.Background(),
		"",
		RetrievalOptions{
			TopK: 5,
		},
	)

	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf(
			"expected ErrEmptyQuery, got %v",
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

func TestRetriever_ReturnsErrorForInvalidTopK(
	t *testing.T,
) {
	retriever := NewRetriever(
		&testEmbedder{},
		NewInMemoryVectorStore(),
	)

	results, err := retriever.Retrieve(
		context.Background(),
		"query",
		RetrievalOptions{
			TopK: 0,
		},
	)

	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf(
			"expected ErrInvalidTopK, got %v",
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

	results, err := retriever.Retrieve(
		context.Background(),
		"database timeout",
		RetrievalOptions{
			TopK: 5,
		},
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

func TestRetriever_AppliesSimilarityThreshold(
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
			ID:      "strong",
			Content: "Database timeout is five seconds.",
			Vector:  []float32{1, 0},
		},
		{
			ID:      "weak",
			Content: "Something vaguely related.",
			Vector:  []float32{0.6, 0.8},
		},
		{
			ID:      "unrelated",
			Content: "UUID generation.",
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

	results, err := retriever.Retrieve(
		context.Background(),
		"database timeout",
		RetrievalOptions{
			TopK:          3,
			MinSimilarity: 0.8,
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf(
			"expected 1 result, got %d",
			len(results),
		)
	}

	if results[0].Document.ID != "strong" {
		t.Fatalf(
			"expected strong document, got %q",
			results[0].Document.ID,
		)
	}
}
