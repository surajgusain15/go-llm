package rag

import (
	"context"
	"errors"
)

var (
	ErrEmptyQuery = errors.New(
		"query cannot be empty",
	)

	ErrInvalidTopK = errors.New(
		"topK must be greater than zero",
	)
)

type Retriever struct {
	embedder Embedder
	store    *InMemoryVectorStore
}

func NewRetriever(
	embedder Embedder,
	store *InMemoryVectorStore,
) *Retriever {
	return &Retriever{
		embedder: embedder,
		store:    store,
	}
}

func (r *Retriever) Retrieve(
	ctx context.Context,
	query string,
	topK int,
) ([]SearchResult, error) {

	if query == "" {
		return nil, ErrEmptyQuery
	}

	if topK <= 0 {
		return nil, ErrInvalidTopK
	}

	vector, err := r.embedder.Embed(
		ctx,
		query,
	)
	if err != nil {
		return nil, err
	}

	return r.store.Search(
		vector,
		topK,
	), nil
}
