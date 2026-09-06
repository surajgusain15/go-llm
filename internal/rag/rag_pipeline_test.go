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

type testGenerator struct {
	answer         string
	capturedPrompt RAGPrompt
	called         bool
	err            error
}

func (g *testGenerator) Generate(
	ctx context.Context,
	prompt RAGPrompt,
) (string, error) {
	g.called = true
	g.capturedPrompt = prompt

	if g.err != nil {
		return "", g.err
	}

	return g.answer, nil
}

func TestRAGPipeline_GenerateCallsGenerator(t *testing.T) {
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

	generator := &testGenerator{
		answer: "Connections should be closed after use.",
	}

	_, err = pipeline.Generate(
		context.Background(),
		"How should database connections be handled?",
		RetrievalOptions{TopK: 1},
		generator,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !generator.called {
		t.Fatal("expected generator to be called")
	}

	if generator.capturedPrompt.Query != "How should database connections be handled?" {
		t.Fatalf(
			"expected generator to receive query, got %q",
			generator.capturedPrompt.Query,
		)
	}
}

func TestRAGPipeline_GenerateReturnsAnswer(t *testing.T) {
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

	generator := &testGenerator{
		answer: "Connections should be closed after use.",
	}

	result, err := pipeline.Generate(
		context.Background(),
		"database",
		RetrievalOptions{TopK: 1},
		generator,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Answer != generator.answer {
		t.Fatalf(
			"expected answer %q, got %q",
			generator.answer,
			result.Answer,
		)
	}
}

func TestRAGPipeline_GeneratePreservesRetrievalCounts(t *testing.T) {
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

	generator := &testGenerator{
		answer: "test answer",
	}

	result, err := pipeline.Generate(
		context.Background(),
		"database",
		RetrievalOptions{TopK: 3},
		generator,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RetrievedCount != 3 {
		t.Fatalf(
			"expected 3 retrieved results, got %d",
			result.RetrievedCount,
		)
	}

	if result.SelectedCount != 2 {
		t.Fatalf(
			"expected 2 selected results, got %d",
			result.SelectedCount,
		)
	}
}

func TestRAGPipeline_GeneratePropagatesGeneratorError(t *testing.T) {
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

	expectedErr := errors.New("generation failed")

	generator := &testGenerator{
		err: expectedErr,
	}

	_, err = pipeline.Generate(
		context.Background(),
		"database",
		RetrievalOptions{TopK: 1},
		generator,
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected error %v, got %v",
			expectedErr,
			err,
		)
	}
}

func TestRAGPipeline_GenerateDoesNotCallGeneratorWhenRetrievalFails(t *testing.T) {
	expectedErr := errors.New("embedding failed")

	embedder := &errorTestEmbedder{
		err: expectedErr,
	}

	store := NewInMemoryVectorStore()
	retriever := NewRetriever(embedder, store)

	budget := NewContextBudget(ApproximateTokenCounter{}, 100)
	ragRetriever := NewRAGRetriever(retriever, budget)
	pipeline := NewRAGPipeline(ragRetriever)

	generator := &testGenerator{
		answer: "should not be returned",
	}

	_, err := pipeline.Generate(
		context.Background(),
		"database",
		RetrievalOptions{TopK: 3},
		generator,
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected error %v, got %v",
			expectedErr,
			err,
		)
	}

	if generator.called {
		t.Fatal("expected generator not to be called")
	}
}
