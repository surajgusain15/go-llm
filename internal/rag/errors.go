package rag

import "errors"

var ErrNoContext = errors.New("no relevant context found")

var ErrInvalidSimilarity = errors.New(
	"minimum similarity must be between -1 and 1",
)
