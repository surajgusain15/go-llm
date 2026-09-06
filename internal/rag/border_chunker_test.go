package rag

import (
	"reflect"
	"testing"
)

func TestBoundaryChunker_PrefersSentenceBoundary(t *testing.T) {
	chunker, err := NewBoundaryChunker(50, 0)
	if err != nil {
		t.Fatal(err)
	}

	text := "Database connections should be closed. Connection pooling improves performance."

	got := chunker.Chunk(text)

	want := []string{
		"Database connections should be closed.",
		"Connection pooling improves performance.",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBoundaryChunker_FallsBackToSizeLimit(t *testing.T) {
	chunker, err := NewBoundaryChunker(10, 0)
	if err != nil {
		t.Fatal(err)
	}

	text := "abcdefghijklmnopqrst"

	got := chunker.Chunk(text)

	want := []string{
		"abcdefghij",
		"klmnopqrst",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBoundaryChunker_PrefersLatestSentenceBoundary(t *testing.T) {
	chunker, err := NewBoundaryChunker(40, 0)
	if err != nil {
		t.Fatal(err)
	}

	text := "First sentence. Second sentence. Third sentence is much longer."

	got := chunker.Chunk(text)

	want := []string{
		"First sentence. Second sentence.",
		"Third sentence is much longer.",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBoundaryChunker_SupportsDifferentSentenceTerminators(t *testing.T) {
	chunker, err := NewBoundaryChunker(30, 0)
	if err != nil {
		t.Fatal(err)
	}

	text := "Is this working? Yes! It is definitely working."

	got := chunker.Chunk(text)

	want := []string{
		"Is this working? Yes!",
		"It is definitely working.",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBoundaryChunker_PreservesOverlap(t *testing.T) {
	chunker, err := NewBoundaryChunker(20, 5)
	if err != nil {
		t.Fatal(err)
	}

	text := "Database connections should be closed."

	got := chunker.Chunk(text)

	if len(got) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d: %#v", len(got), got)
	}

	// The second chunk should contain the tail of the first chunk.
	first := got[0]
	second := got[1]

	overlap := first[len(first)-5:]

	if len(second) < 5 || second[:5] != overlap {
		t.Fatalf(
			"expected second chunk to begin with overlap %q, got %q",
			overlap,
			second,
		)
	}
}

func TestBoundaryChunker_HandlesUnicode(t *testing.T) {
	chunker, err := NewBoundaryChunker(10, 0)
	if err != nil {
		t.Fatal(err)
	}

	text := "你好世界，这是一个测试。"

	got := chunker.Chunk(text)

	if len(got) == 0 {
		t.Fatal("expected chunks, got none")
	}

	for _, chunk := range got {
		if len([]rune(chunk)) > 10 {
			t.Fatalf(
				"chunk exceeds size limit: %q (%d runes)",
				chunk,
				len([]rune(chunk)),
			)
		}
	}
}

func TestBoundaryChunker_EmptyText(t *testing.T) {
	chunker, err := NewBoundaryChunker(10, 2)
	if err != nil {
		t.Fatal(err)
	}

	got := chunker.Chunk("")

	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestBoundaryChunker_InvalidSize(t *testing.T) {
	_, err := NewBoundaryChunker(0, 0)

	if err != ErrInvalidChunkSize {
		t.Fatalf("expected %v, got %v", ErrInvalidChunkSize, err)
	}
}

func TestBoundaryChunker_InvalidOverlap(t *testing.T) {
	_, err := NewBoundaryChunker(10, 10)

	if err != ErrInvalidChunkOverlap {
		t.Fatalf("expected %v, got %v", ErrInvalidChunkOverlap, err)
	}
}

func TestBoundaryChunker_NegativeOverlap(t *testing.T) {
	_, err := NewBoundaryChunker(10, -1)

	if err != ErrInvalidChunkOverlap {
		t.Fatalf("expected %v, got %v", ErrInvalidChunkOverlap, err)
	}
}
