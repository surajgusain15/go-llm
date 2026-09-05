package rag

import (
	"context"
	"testing"
)

type RetrievalTestCase struct {
	Query          string
	RelevantDocIDs []string
}

func TestRetriever_RecallAtK(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database timeout": {1, 0},
			"database latency": {0.9, 0.1},
			"uuid generation":  {0, 1},
		},
	}

	store := NewInMemoryVectorStore()

	documents := []Document{
		{
			ID:      "timeout",
			Content: "Database connection timeout is five seconds.",
			Vector:  []float32{1, 0},
		},
		{
			ID:      "latency",
			Content: "Database latency is measured in milliseconds.",
			Vector:  []float32{0.9, 0.1},
		},
		{
			ID:      "uuid",
			Content: "UUID generation is supported.",
			Vector:  []float32{0, 1},
		},
	}

	for _, document := range documents {
		if err := store.Add(document); err != nil {
			t.Fatal(err)
		}
	}

	retriever := NewRetriever(embedder, store)

	testCases := []RetrievalTestCase{
		{
			Query:          "database timeout",
			RelevantDocIDs: []string{"timeout"},
		},
		{
			Query:          "database latency",
			RelevantDocIDs: []string{"latency"},
		},
		{
			Query:          "uuid generation",
			RelevantDocIDs: []string{"uuid"},
		},
	}

	const topK = 2

	for _, testCase := range testCases {
		t.Run(
			testCase.Query, func(t *testing.T) {
				results, err := retriever.Retrieve(
					context.Background(),
					testCase.Query,
					topK,
				)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if recall := recallAtK(results, testCase.RelevantDocIDs); recall < 1.0 {
					t.Fatalf(
						"expected Recall@%d = 1.0, got %.2f",
						topK,
						recall,
					)
				}
			},
		)
	}
}

func recallAtK(
	results []SearchResult,
	relevantDocIDs []string,
) float64 {
	if len(relevantDocIDs) == 0 {
		return 0
	}

	relevant := make(map[string]struct{}, len(relevantDocIDs))

	for _, id := range relevantDocIDs {
		relevant[id] = struct{}{}
	}

	found := 0

	for _, result := range results {
		if _, ok := relevant[result.Document.ID]; ok {
			found++
		}
	}

	return float64(found) / float64(len(relevantDocIDs))
}

func TestRetriever_PrecisionAtK(t *testing.T) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"database timeout": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	documents := []Document{
		{
			ID:      "timeout",
			Content: "Database connection timeout is five seconds.",
			Vector:  []float32{1, 0},
		},
		{
			ID:      "latency",
			Content: "Database latency is measured in milliseconds.",
			Vector:  []float32{0.9, 0.1},
		},
		{
			ID:      "uuid",
			Content: "UUID generation is supported.",
			Vector:  []float32{0, 1},
		},
	}

	for _, document := range documents {
		if err := store.Add(document); err != nil {
			t.Fatal(err)
		}
	}

	retriever := NewRetriever(embedder, store)

	results, err := retriever.Retrieve(
		context.Background(),
		"database timeout",
		2,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	precision := precisionAtK(
		results,
		[]string{"timeout"},
	)

	expected := 0.5

	if precision != expected {
		t.Fatalf(
			"expected Precision@2 = %.2f, got %.2f",
			expected,
			precision,
		)
	}
}

func precisionAtK(
	results []SearchResult,
	relevantDocIDs []string,
) float64 {
	if len(results) == 0 {
		return 0
	}

	relevant := make(map[string]struct{}, len(relevantDocIDs))

	for _, id := range relevantDocIDs {
		relevant[id] = struct{}{}
	}

	found := 0

	for _, result := range results {
		if _, ok := relevant[result.Document.ID]; ok {
			found++
		}
	}

	return float64(found) / float64(len(results))
}
