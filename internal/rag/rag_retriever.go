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

// func (r *RAGRetriever) Retrieve(
// 	ctx context.Context,
// 	query string,
// 	options RetrievalOptions,
// ) (Context, error) {
// 	results, err := r.retriever.Retrieve(ctx, query, options)
// 	if err != nil {
// 		return Context{}, err
// 	}
//
// 	selected := r.contextBudget.Select(results)
//
// 	return NewContext(selected), nil
// }

type RetrievalContext struct {
	Context        Context
	RetrievedCount int
	SelectedCount  int
}

func (r *RAGRetriever) Retrieve(
	ctx context.Context,
	query string,
	options RetrievalOptions,
) (RetrievalContext, error) {
	results, err := r.retriever.Retrieve(ctx, query, options)
	if err != nil {
		return RetrievalContext{}, err
	}

	selected := r.contextBudget.Select(results)

	return RetrievalContext{
		Context:        NewContext(selected),
		RetrievedCount: len(results),
		SelectedCount:  len(selected),
	}, nil
}
