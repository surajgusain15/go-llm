package rag

import "context"

type RAGPipeline struct {
	retriever *RAGRetriever
}

func NewRAGPipeline(retriever *RAGRetriever) *RAGPipeline {
	return &RAGPipeline{
		retriever: retriever,
	}
}

type RAGResult struct {
	Prompt         RAGPrompt
	Answer         string
	RetrievedCount int
	SelectedCount  int
}

func (p *RAGPipeline) BuildPrompt(
	ctx context.Context,
	query string,
	options RetrievalOptions,
) (RAGResult, error) {
	retrieval, err := p.retriever.Retrieve(ctx, query, options)
	if err != nil {
		return RAGResult{}, err
	}

	return RAGResult{
		Prompt:         NewRAGPrompt(query, retrieval.Context),
		RetrievedCount: retrieval.RetrievedCount,
		SelectedCount:  retrieval.SelectedCount,
	}, nil
}

func (p *RAGPipeline) Generate(
	ctx context.Context,
	query string,
	options RetrievalOptions,
	generator Generator,
) (RAGResult, error) {
	result, err := p.BuildPrompt(ctx, query, options)
	if err != nil {
		return RAGResult{}, err
	}

	if result.SelectedCount == 0 {
		return result, ErrNoContext
	}

	answer, err := generator.Generate(ctx, result.Prompt)
	if err != nil {
		return RAGResult{}, err
	}

	result.Answer = answer

	return result, nil
}
