package rag

import (
	"context"

	"go-llm/internal/llm"
)

type EmbedderAdapter struct {
	embedder llm.Embedder
}

func NewEmbedderAdapter(embedder llm.Embedder) *EmbedderAdapter {
	return &EmbedderAdapter{
		embedder: embedder,
	}
}

func (a *EmbedderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return a.embedder.Embed(ctx, text)
}
