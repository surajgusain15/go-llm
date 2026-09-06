package rag

import (
	"context"
	"testing"
)

type testLLMGenerator struct {
	prompt string
	answer string
	err    error
}

func (g *testLLMGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	g.prompt = prompt

	if g.err != nil {
		return "", g.err
	}

	return g.answer, nil
}

func TestGeneratorAdapter_ForwardsPrompt(t *testing.T) {
	provider := &testLLMGenerator{
		answer: "connections should be closed",
	}

	adapter := NewGeneratorAdapter(provider)

	prompt := NewRAGPrompt(
		"What should happen to database connections?",
		NewContext(
			[]SearchResult{
				{
					Document: Document{
						ID:      "doc-1",
						Content: "Database connections should be closed.",
					},
				},
			},
		),
	)

	answer, err := adapter.Generate(context.Background(), prompt)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if answer != "connections should be closed" {
		t.Fatalf("answer = %q", answer)
	}

	if provider.prompt != prompt.String() {
		t.Fatalf("provider prompt = %q, want %q", provider.prompt, prompt.String())
	}
}
