package rag

import (
	"errors"
	"slices"
	"strings"
)

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

type BoundaryChunker struct {
	size    int
	overlap int
}

func NewBoundaryChunker(
	size int,
	overlap int,
) (*BoundaryChunker, error) {
	if size <= 0 {
		return nil, ErrInvalidChunkSize
	}

	if overlap < 0 || overlap >= size {
		return nil, ErrInvalidChunkOverlap
	}

	return &BoundaryChunker{
		size:    size,
		overlap: overlap,
	}, nil
}

func (c *BoundaryChunker) Chunk(text string) []string {
	if text == "" {
		return nil
	}

	runes := []rune(text)

	if len(runes) <= c.size {
		return []string{strings.TrimSpace(text)}
	}

	var chunks []string

	start := 0

	for start < len(runes) {
		end := start + c.size

		if end >= len(runes) {
			chunk := strings.TrimSpace(
				string(runes[start:]),
			)

			if chunk != "" {
				chunks = append(chunks, chunk)
			}

			break
		}

		window := runes[start:end]

		cut := c.findBoundary(window)

		if cut == len(window) {
			// No sentence boundary found.
			cut = len(window)
		}

		chunk := strings.TrimSpace(
			string(runes[start : start+cut]),
		)

		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		nextStart := start + cut - c.overlap

		if nextStart <= start {
			nextStart = start + cut
		}

		start = nextStart
	}

	return chunks
}

func (c *BoundaryChunker) findBoundary(
	runes []rune,
) int {
	for i, r := range slices.Backward(runes) {
		switch r {
		case '.', '!', '?':
			return i + 1
		}
	}

	return len(runes)
}
