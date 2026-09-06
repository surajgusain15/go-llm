package rag

import "strings"

type Context struct {
	Chunks []SearchResult
}

func NewContext(results []SearchResult) Context {
	return Context{
		Chunks: results,
	}
}

func (c Context) Text() string {
	if len(c.Chunks) == 0 {
		return ""
	}

	var builder strings.Builder

	for index, result := range c.Chunks {
		if index > 0 {
			builder.WriteString("\n\n")
		}

		builder.WriteString(result.Document.Content)
	}

	return builder.String()
}
