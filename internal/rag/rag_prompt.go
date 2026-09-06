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
If the answer cannot be found in the context, say that the information is not available in the provided context.
Do not use outside knowledge or make up information.

Context:
%s

Question:
%s`,
		p.Context.Text(),
		p.Query,
	)
}
