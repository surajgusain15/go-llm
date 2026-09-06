package rag

import "fmt"

type RAGPrompt struct {
	Query   string
	Context Context
}

func NewRAGPrompt(query string, context Context) RAGPrompt {
	return RAGPrompt{
		Query:   query,
		Context: context,
	}
}

func (p RAGPrompt) Text() string {
	return fmt.Sprintf(
		`Answer the question using only the provided context.

Context:
%s

Question:
%s`,
		p.Context.Text(),
		p.Query,
	)
}
