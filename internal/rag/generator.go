package rag

import "context"

type Generator interface {
	Generate(ctx context.Context, prompt RAGPrompt) (string, error)
}
