package rag

type ContextBudgetConfig struct {
	ModelContextTokens     int
	SystemPromptTokens     int
	ConversationTokens     int
	QueryTokens            int
	ReservedResponseTokens int
}

func (c ContextBudgetConfig) RetrievalBudget() int {
	budget := c.ModelContextTokens -
		c.SystemPromptTokens -
		c.ConversationTokens -
		c.QueryTokens -
		c.ReservedResponseTokens

	if budget <= 0 {
		return 0
	}

	return budget
}
