package llm

import "context"

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
