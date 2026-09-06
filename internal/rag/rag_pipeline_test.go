package rag

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRAGPipeline_BuildPromptIncludesQuery(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	err := store.Add(
		Document{
			ID:      "doc-1",
			Content: "Database connections should be closed.",
			Vector:  []float32{1, 0},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error adding document: %v", err)
	}

	retriever := NewRetriever(embedder, store)
	budget := NewContextBudget(ApproximateTokenCounter{}, 100)
	ragRetriever := NewRAGRetriever(retriever, budget)
	pipeline := NewRAGPipeline(ragRetriever)

	query := "How should database connections be handled?"

	result, err := pipeline.BuildPrompt(
		context.Background(),
		query,
		RetrievalOptions{TopK: 1},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Prompt.Text(), query) {
		t.Fatalf("expected prompt to contain query, got %q", result.Prompt.Text())
	}
}

func TestRAGPipeline_BuildPromptIncludesRetrievedContext(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	err := store.Add(
		Document{
			ID:      "doc-1",
			Content: "Database connections should be closed after every request.",
			Vector:  []float32{1, 0},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error adding document: %v", err)
	}

	retriever := NewRetriever(embedder, store)
	budget := NewContextBudget(ApproximateTokenCounter{}, 100)
	ragRetriever := NewRAGRetriever(retriever, budget)
	pipeline := NewRAGPipeline(ragRetriever)

	result, err := pipeline.BuildPrompt(
		context.Background(),
		"database",
		RetrievalOptions{TopK: 1},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Database connections should be closed after every request."

	if !strings.Contains(result.Prompt.Text(), expected) {
		t.Fatalf("expected prompt to contain retrieved context, got %q", result.Prompt.Text())
	}
}

func TestRAGPipeline_BuildPromptPreservesRetrievalCounts(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	for _, document := range []Document{
		{
			ID:      "doc-1",
			Content: "12345678",
			Vector:  []float32{1, 0},
		},
		{
			ID:      "doc-2",
			Content: "abcdefgh",
			Vector:  []float32{1, 0},
		},
		{
			ID:      "doc-3",
			Content: "ijklmnop",
			Vector:  []float32{1, 0},
		},
	} {
		if err := store.Add(document); err != nil {
			t.Fatalf("unexpected error adding document: %v", err)
		}
	}

	retriever := NewRetriever(embedder, store)
	budget := NewContextBudget(ApproximateTokenCounter{}, 5)
	ragRetriever := NewRAGRetriever(retriever, budget)
	pipeline := NewRAGPipeline(ragRetriever)

	result, err := pipeline.BuildPrompt(
		context.Background(),
		"database",
		RetrievalOptions{TopK: 3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RetrievedCount != 3 {
		t.Fatalf("expected 3 retrieved results, got %d", result.RetrievedCount)
	}

	if result.SelectedCount == 0 {
		t.Fatal("expected at least one selected result")
	}

	if result.SelectedCount >= result.RetrievedCount {
		t.Fatalf(
			"expected context budget to select fewer results than retrieved, got %d selected out of %d",
			result.SelectedCount,
			result.RetrievedCount,
		)
	}
}

func TestRAGPipeline_PropagatesRetrievalError(t *testing.T) {
	expectedErr := errors.New("embedding failed")

	embedder := &errorTestEmbedder{
		err: expectedErr,
	}

	store := NewInMemoryVectorStore()
	retriever := NewRetriever(embedder, store)

	budget := NewContextBudget(ApproximateTokenCounter{}, 100)
	ragRetriever := NewRAGRetriever(retriever, budget)
	pipeline := NewRAGPipeline(ragRetriever)

	_, err := pipeline.BuildPrompt(
		context.Background(),
		"database",
		RetrievalOptions{TopK: 3},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
