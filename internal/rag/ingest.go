package rag

import (
	"context"
	"errors"
)

var (
	ErrEmptyDocumentContent = errors.New("document content cannot be empty")
)

type Ingestor struct {
	embedder Embedder
	store    *InMemoryVectorStore
}

func NewIngestor(
	embedder Embedder,
	store *InMemoryVectorStore,
) *Ingestor {
	return &Ingestor{
		embedder: embedder,
		store:    store,
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

	vector, err := i.embedder.Embed(
		ctx,
		document.Content,
	)
	if err != nil {
		return err
	}

	document.Vector = vector

	return i.store.Add(document)
}
