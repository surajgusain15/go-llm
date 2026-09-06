package rag

import "testing"

func TestContextBudgetConfig_RetrievalBudget(t *testing.T) {
	config := ContextBudgetConfig{
		ModelContextTokens:     8192,
		SystemPromptTokens:     500,
		ConversationTokens:     1000,
		QueryTokens:            100,
		ReservedResponseTokens: 1500,
	}

	got := config.RetrievalBudget()

	want := 5092

	if got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
}

func TestContextBudgetConfig_ReturnsZeroWhenBudgetIsExhausted(t *testing.T) {
	config := ContextBudgetConfig{
		ModelContextTokens:     1000,
		SystemPromptTokens:     500,
		ConversationTokens:     300,
		QueryTokens:            100,
		ReservedResponseTokens: 100,
	}

	got := config.RetrievalBudget()

	if got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestContextBudgetConfig_ReturnsZeroWhenBudgetIsNegative(t *testing.T) {
	config := ContextBudgetConfig{
		ModelContextTokens:     1000,
		SystemPromptTokens:     800,
		ConversationTokens:     500,
		QueryTokens:            100,
		ReservedResponseTokens: 100,
	}

	got := config.RetrievalBudget()

	if got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestContextBudgetConfig_AllowsEntireContextForRetrieval(t *testing.T) {
	config := ContextBudgetConfig{
		ModelContextTokens: 1000,
	}

	got := config.RetrievalBudget()

	if got != 1000 {
		t.Fatalf("expected 1000, got %d", got)
	}
}
