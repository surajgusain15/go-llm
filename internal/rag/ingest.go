package rag

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrEmptyDocumentContent = errors.New(
		"document content cannot be empty",
	)
)

type Ingestor struct {
	embedder Embedder
	store    *InMemoryVectorStore
	chunker  *Chunker
}

func NewIngestor(
	embedder Embedder,
	store *InMemoryVectorStore,
	chunker *Chunker,
) *Ingestor {
	return &Ingestor{
		embedder: embedder,
		store:    store,
		chunker:  chunker,
	}
}

func (i *Ingestor) Ingest(
	ctx context.Context,
	document Document,
) error {

	if document.ID == "" {
		return ErrEmptyDocumentID
	}

	if document.Content == "" {
		return ErrEmptyDocumentContent
	}

	chunks := i.chunker.Chunk(
		document.Content,
	)

	for index, chunk := range chunks {

		vector, err := i.embedder.Embed(
			ctx,
			chunk,
		)
		if err != nil {
			return err
		}

		chunkDocument := Document{
			ID: fmt.Sprintf(
				"%s#chunk-%d",
				document.ID,
				index,
			),
			Content: chunk,
			Vector:  vector,
		}

		if err := i.store.Add(
			chunkDocument,
		); err != nil {
			return err
		}
	}

	return nil
}
