package rag

type ContextBudget struct {
	counter TokenCounter
	limit   int
}

func NewContextBudget(counter TokenCounter, limit int) *ContextBudget {
	return &ContextBudget{
		counter: counter,
		limit:   limit,
	}
}

func (b *ContextBudget) Select(results []SearchResult) []SearchResult {
	if b.limit <= 0 {
		return nil
	}

	selected := make([]SearchResult, 0, len(results))
	used := 0

	for _, result := range results {
		tokens := b.counter.Count(result.Document.Content)

		if used+tokens > b.limit {
			break
		}

		selected = append(selected, result)
		used += tokens
	}

	return selected
}
