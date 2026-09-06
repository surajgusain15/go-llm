package llm

import "context"

type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
