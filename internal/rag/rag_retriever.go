package rag

import "context"

type RAGRetriever struct {
	retriever     *Retriever
	contextBudget *ContextBudget
}

func NewRAGRetriever(
	retriever *Retriever,
	contextBudget *ContextBudget,
) *RAGRetriever {
	return &RAGRetriever{
		retriever:     retriever,
		contextBudget: contextBudget,
	}
}

func (r *RAGRetriever) Retrieve(
	ctx context.Context,
	query string,
	options RetrievalOptions,
) (Context, error) {
	results, err := r.retriever.Retrieve(ctx, query, options)
	if err != nil {
		return Context{}, err
	}

	selected := r.contextBudget.Select(results)

	return NewContext(selected), nil
}
