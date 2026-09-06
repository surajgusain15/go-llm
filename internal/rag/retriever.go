package rag

import (
	"context"
	"errors"
)

var (
	ErrEmptyQuery  = errors.New("query cannot be empty")
	ErrInvalidTopK = errors.New("topK must be greater than zero")
)

type RetrievalOptions struct {
	TopK          int
	MinSimilarity float32
}

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
	options RetrievalOptions,
) ([]SearchResult, error) {
	if query == "" {
		return nil, ErrEmptyQuery
	}

	if options.TopK <= 0 {
		return nil, ErrInvalidTopK
	}

	vector, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	return r.store.SearchWithThreshold(
		vector,
		options.TopK,
		options.MinSimilarity,
	), nil
}
