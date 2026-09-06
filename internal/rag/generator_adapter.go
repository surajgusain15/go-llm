package rag

import (
	"context"

	"go-llm/internal/llm"
)

type GeneratorAdapter struct {
	generator llm.Generator
}

func NewGeneratorAdapter(generator llm.Generator) *GeneratorAdapter {
	return &GeneratorAdapter{
		generator: generator,
	}
}

func (a *GeneratorAdapter) Generate(ctx context.Context, prompt RAGPrompt) (string, error) {
	return a.generator.Generate(ctx, prompt.String())
}
