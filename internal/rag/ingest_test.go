package rag

import (
	"context"
	"errors"
	"testing"
)

func TestIngestor_EmbedsAndStoresDocument(
	t *testing.T,
) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"provider has low latency": {1, 0},
		},
	}

	store := NewInMemoryVectorStore()

	ingestor := NewIngestor(
		embedder,
		store,
	)

	document := Document{
		ID:      "provider-1",
		Content: "provider has low latency",
	}

	err := ingestor.Ingest(
		context.Background(),
		document,
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	results := store.Search(
		[]float32{1, 0},
		1,
	)

	if len(results) != 1 {
		t.Fatalf(
			"expected 1 result, got %d",
			len(results),
		)
	}

	if results[0].Document.ID != "provider-1" {
		t.Fatalf(
			"expected provider-1, got %q",
			results[0].Document.ID,
		)
	}

	if len(results[0].Document.Vector) != 2 {
		t.Fatalf(
			"expected 2-dimensional vector, got %d",
			len(results[0].Document.Vector),
		)
	}
}

func TestIngestor_RejectsEmptyContent(
	t *testing.T,
) {
	ingestor := NewIngestor(
		&testEmbedder{},
		NewInMemoryVectorStore(),
	)

	err := ingestor.Ingest(
		context.Background(),
		Document{
			ID: "provider-1",
		},
	)

	if !errors.Is(
		err,
		ErrEmptyDocumentContent,
	) {
		t.Fatalf(
			"expected ErrEmptyDocumentContent, got %v",
			err,
		)
	}
}

func TestIngestor_PropagatesEmbeddingError(
	t *testing.T,
) {
	expectedErr := errors.New("embedding failed")

	embedder := &failingEmbedder{
		err: expectedErr,
	}

	store := NewInMemoryVectorStore()

	ingestor := NewIngestor(
		embedder,
		store,
	)

	err := ingestor.Ingest(
		context.Background(),
		Document{
			ID:      "provider-1",
			Content: "provider has low latency",
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected embedding error, got %v",
			err,
		)
	}

	results := store.Search(
		[]float32{1, 0},
		10,
	)

	if len(results) != 0 {
		t.Fatalf(
			"expected document not to be stored, got %d",
			len(results),
		)
	}
}

type failingEmbedder struct {
	err error
}

func (e *failingEmbedder) Embed(
	ctx context.Context,
	text string,
) ([]float32, error) {
	return nil, e.err
}
