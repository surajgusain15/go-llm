package rag

import "context"

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

func (r *Retriever) Search(
	ctx context.Context,
	query string,
	topK int,
) ([]SearchResult, error) {

	if query == "" || topK <= 0 {
		return nil, nil
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
