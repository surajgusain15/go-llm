package rag

import "errors"

var (
	ErrInvalidChunkSize    = errors.New("chunk size must be greater than zero")
	ErrInvalidChunkOverlap = errors.New(
		"chunk overlap must be greater than or equal to zero and smaller than chunk size",
	)
)

type Chunker struct {
	size    int
	overlap int
}

func NewChunker(
	size int,
	overlap int,
) (*Chunker, error) {

	if size <= 0 {
		return nil, ErrInvalidChunkSize
	}

	if overlap < 0 || overlap >= size {
		return nil, ErrInvalidChunkOverlap
	}

	return &Chunker{
		size:    size,
		overlap: overlap,
	}, nil
}

func (c *Chunker) Chunk(
	text string,
) []string {

	if text == "" {
		return nil
	}

	var chunks []string

	step := c.size - c.overlap

	for start := 0; start < len(text); start += step {

		end := start + c.size

		if end > len(text) {
			end = len(text)
		}

		chunks = append(
			chunks,
			text[start:end],
		)

		if end == len(text) {
			break
		}
	}

	return chunks
}
