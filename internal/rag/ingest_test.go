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

	chunker, err := NewChunker(100, 20)
	if err != nil {
		t.Fatal(err)
	}

	ingestor := NewIngestor(
		embedder,
		store,
		chunker,
	)

	document := Document{
		ID:      "provider-1",
		Content: "provider has low latency",
	}

	err = ingestor.Ingest(
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

	if results[0].Document.ID != "provider-1#chunk-0" {
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

	chunker, err := NewChunker(100, 20)
	if err != nil {
		t.Fatal(err)
	}

	ingestor := NewIngestor(
		&testEmbedder{},
		NewInMemoryVectorStore(),
		chunker,
	)

	err = ingestor.Ingest(
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

	chunker, err := NewChunker(100, 20)
	if err != nil {
		t.Fatal(err)
	}

	ingestor := NewIngestor(
		embedder,
		store,
		chunker,
	)

	err = ingestor.Ingest(
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

func TestIngestor_ChunksEmbedsAndStores(
	t *testing.T,
) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"abcdefghij": {1, 0},
			"ijqrstuvwx": {0, 1},
			"wx":         {1, 1},
		},
	}

	store := NewInMemoryVectorStore()

	chunker, err := NewChunker(10, 2)
	if err != nil {
		t.Fatal(err)
	}

	ingestor := NewIngestor(
		embedder,
		store,
		chunker,
	)

	err = ingestor.Ingest(
		context.Background(),
		Document{
			ID:      "document-1",
			Content: "abcdefghijqrstuvwx",
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	results := store.Search(
		[]float32{1, 0},
		10,
	)

	expected := []struct {
		id      string
		content string
	}{
		{
			id:      "document-1#chunk-0",
			content: "abcdefghij",
		},
		{
			id:      "document-1#chunk-1",
			content: "ijqrstuvwx",
		},
	}

	if len(results) != len(expected) {
		t.Fatalf(
			"expected %d chunks, got %d",
			len(expected),
			len(results),
		)
	}
}

func TestIngestor_StoresChunkMetadata(
	t *testing.T,
) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"abcdefghij": {1, 0},
			"ijqrstuvwx": {0, 1},
		},
	}

	store := NewInMemoryVectorStore()

	chunker, err := NewChunker(10, 2)
	if err != nil {
		t.Fatal(err)
	}

	ingestor := NewIngestor(
		embedder,
		store,
		chunker,
	)

	err = ingestor.Ingest(
		context.Background(),
		Document{
			ID:      "document-1",
			Content: "abcdefghijqrstuvwx",
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	results := store.Search(
		[]float32{1, 0},
		10,
	)

	if len(results) != 2 {
		t.Fatalf(
			"expected 2 chunks, got %d",
			len(results),
		)
	}

	for _, result := range results {
		metadata := result.Document.Metadata

		if metadata.SourceDocumentID != "document-1" {
			t.Fatalf(
				"expected source document ID %q, got %q",
				"document-1",
				metadata.SourceDocumentID,
			)
		}

		if metadata.ChunkIndex < 0 ||
			metadata.ChunkIndex > 1 {
			t.Fatalf(
				"unexpected chunk index: %d",
				metadata.ChunkIndex,
			)
		}
	}
}
