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

func TestChunker_CanSplitSentenceAcrossChunks(
	t *testing.T,
) {
	chunker, err := NewChunker(20, 0)

	if err != nil {
		t.Fatal(err)
	}

	text := "Database connections should be closed after every request."

	chunks := chunker.Chunk(text)

	if len(chunks) != 3 {
		t.Fatalf(
			"expected 3 chunks, got %d",
			len(chunks),
		)
	}

	expected := []string{
		"Database connections",
		" should be closed af",
		"ter every request.",
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

func TestBoundaryChunker_DoesNotExceedMaximumSize(
	t *testing.T,
) {
	chunker, err := NewBoundaryChunker(20, 0)

	if err != nil {
		t.Fatal(err)
	}

	text := "This sentence is definitely longer than twenty characters."

	chunks := chunker.Chunk(text)

	for i, chunk := range chunks {
		if len([]rune(chunk)) > 20 {
			t.Fatalf(
				"chunk %d exceeds maximum size: %d",
				i,
				len([]rune(chunk)),
			)
		}
	}
}

func TestBoundaryChunker_SupportsUnicode(
	t *testing.T,
) {
	chunker, err := NewBoundaryChunker(10, 0)

	if err != nil {
		t.Fatal(err)
	}

	text := "Database परीक्षण works. दूसरा sentence works."

	chunks := chunker.Chunk(text)

	for i, chunk := range chunks {
		if len([]rune(chunk)) > 10 {
			t.Fatalf(
				"chunk %d exceeds maximum rune size: %d",
				i,
				len([]rune(chunk)),
			)
		}
	}
}
