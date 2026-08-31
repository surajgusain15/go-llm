package rag

import (
	"math"
	"testing"
)

func TestInMemoryVectorStore_AddAndSearch(
	t *testing.T,
) {
	store := NewInMemoryVectorStore()

	documents := []Document{
		{
			ID:      "timeout",
			Content: "Provider X has a two second timeout.",
			Vector:  []float32{1, 0},
		},
		{
			ID:      "routing",
			Content: "Provider X supports Airtel recharge.",
			Vector:  []float32{0, 1},
		},
		{
			ID:      "latency",
			Content: "Provider X has low latency.",
			Vector:  []float32{0.9, 0.1},
		},
	}

	for _, document := range documents {
		if err := store.Add(document); err != nil {
			t.Fatalf(
				"failed to add %q: %v",
				document.ID,
				err,
			)
		}
	}

	results := store.Search(
		[]float32{1, 0},
		2,
	)

	if len(results) != 2 {
		t.Fatalf(
			"expected 2 results, got %d",
			len(results),
		)
	}

	if results[0].Document.ID != "timeout" {
		t.Fatalf(
			"expected timeout as first result, got %q",
			results[0].Document.ID,
		)
	}

	if results[1].Document.ID != "latency" {
		t.Fatalf(
			"expected latency as second result, got %q",
			results[1].Document.ID,
		)
	}

	if math.Abs(
		float64(results[0].Similarity-1),
	) > 0.00001 {
		t.Fatalf(
			"expected first similarity to be 1, got %v",
			results[0].Similarity,
		)
	}
}

func TestInMemoryVectorStore_TopKGreaterThanDocuments(
	t *testing.T,
) {
	store := NewInMemoryVectorStore()

	err := store.Add(
		Document{
			ID:      "one",
			Content: "one",
			Vector:  []float32{1, 0},
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results := store.Search(
		[]float32{1, 0},
		10,
	)

	if len(results) != 1 {
		t.Fatalf(
			"expected 1 result, got %d",
			len(results),
		)
	}
}

func TestInMemoryVectorStore_RejectsEmptyDocumentID(
	t *testing.T,
) {
	store := NewInMemoryVectorStore()

	err := store.Add(
		Document{
			Content: "test",
			Vector:  []float32{1, 0},
		},
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInMemoryVectorStore_RejectsEmptyVector(
	t *testing.T,
) {
	store := NewInMemoryVectorStore()

	err := store.Add(
		Document{
			ID:      "test",
			Content: "test",
		},
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInMemoryVectorStore_SearchEmptyQuery(
	t *testing.T,
) {
	store := NewInMemoryVectorStore()

	results := store.Search(
		nil,
		3,
	)

	if results != nil {
		t.Fatalf(
			"expected nil results, got %v",
			results,
		)
	}
}

func TestInMemoryVectorStore_SearchZeroTopK(
	t *testing.T,
) {
	store := NewInMemoryVectorStore()

	results := store.Search(
		[]float32{1, 0},
		0,
	)

	if results != nil {
		t.Fatalf(
			"expected nil results, got %v",
			results,
		)
	}
}

func TestInMemoryVectorStore_SkipsIncompatibleVectors(
	t *testing.T,
) {
	store := NewInMemoryVectorStore()

	if err := store.Add(
		Document{
			ID:     "valid",
			Vector: []float32{1, 0},
		},
	); err != nil {
		t.Fatal(err)
	}

	if err := store.Add(
		Document{
			ID:     "wrong-dimension",
			Vector: []float32{1, 0, 0},
		},
	); err != nil {
		t.Fatal(err)
	}

	results := store.Search(
		[]float32{1, 0},
		10,
	)

	if len(results) != 1 {
		t.Fatalf(
			"expected 1 compatible result, got %d",
			len(results),
		)
	}

	if results[0].Document.ID != "valid" {
		t.Fatalf(
			"expected valid document, got %q",
			results[0].Document.ID,
		)
	}
}
