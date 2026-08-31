package rag

import (
	"errors"
	"testing"
)

func TestChunker_SplitsTextWithOverlap(
	t *testing.T,
) {
	chunker, err := NewChunker(10, 2)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	chunks := chunker.Chunk(
		"01234567890123456789",
	)

	expected := []string{
		"0123456789",
		"8901234567",
		"6789",
	}

	if len(chunks) != len(expected) {
		t.Fatalf(
			"expected %d chunks, got %d",
			len(expected),
			len(chunks),
		)
	}

	for i := range expected {
		if chunks[i] != expected[i] {
			t.Fatalf(
				"chunk %d: expected %q, got %q",
				i,
				expected[i],
				chunks[i],
			)
		}
	}
}

func TestChunker_DoesNotCreateEmptyChunks(
	t *testing.T,
) {
	chunker, err := NewChunker(5, 1)

	if err != nil {
		t.Fatal(err)
	}

	chunks := chunker.Chunk("abc")

	if len(chunks) != 1 {
		t.Fatalf(
			"expected 1 chunk, got %d",
			len(chunks),
		)
	}

	if chunks[0] != "abc" {
		t.Fatalf(
			"expected %q, got %q",
			"abc",
			chunks[0],
		)
	}
}

func TestChunker_EmptyText(
	t *testing.T,
) {
	chunker, err := NewChunker(10, 2)

	if err != nil {
		t.Fatal(err)
	}

	chunks := chunker.Chunk("")

	if chunks != nil {
		t.Fatalf(
			"expected nil chunks, got %v",
			chunks,
		)
	}
}

func TestNewChunker_RejectsInvalidSize(
	t *testing.T,
) {
	_, err := NewChunker(0, 0)

	if !errors.Is(err, ErrInvalidChunkSize) {
		t.Fatalf(
			"expected ErrInvalidChunkSize, got %v",
			err,
		)
	}
}

func TestNewChunker_RejectsNegativeOverlap(
	t *testing.T,
) {
	_, err := NewChunker(10, -1)

	if !errors.Is(err, ErrInvalidChunkOverlap) {
		t.Fatalf(
			"expected ErrInvalidChunkOverlap, got %v",
			err,
		)
	}
}

func TestNewChunker_RejectsOverlapEqualToSize(
	t *testing.T,
) {
	_, err := NewChunker(10, 10)

	if !errors.Is(err, ErrInvalidChunkOverlap) {
		t.Fatalf(
			"expected ErrInvalidChunkOverlap, got %v",
			err,
		)
	}
}

func TestNewChunker_RejectsOverlapGreaterThanSize(
	t *testing.T,
) {
	_, err := NewChunker(10, 11)

	if !errors.Is(err, ErrInvalidChunkOverlap) {
		t.Fatalf(
			"expected ErrInvalidChunkOverlap, got %v",
			err,
		)
	}
}
