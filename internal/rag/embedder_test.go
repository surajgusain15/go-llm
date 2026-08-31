package rag

import (
	"context"
	"testing"
)

type testEmbedder struct {
	embeddings map[string][]float32
}

func (e *testEmbedder) Embed(
	ctx context.Context,
	text string,
) ([]float32, error) {

	vector := e.embeddings[text]

	return append(
		[]float32(nil),
		vector...,
	), nil
}

func TestEmbedder_ReturnsEmbedding(
	t *testing.T,
) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"hello": {1, 2, 3},
		},
	}

	got, err := embedder.Embed(
		context.Background(),
		"hello",
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	want := []float32{1, 2, 3}

	if len(got) != len(want) {
		t.Fatalf(
			"expected %d dimensions, got %d",
			len(want),
			len(got),
		)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf(
				"dimension %d: expected %v, got %v",
				i,
				want[i],
				got[i],
			)
		}
	}
}

func TestEmbedder_DoesNotExposeInternalVector(
	t *testing.T,
) {
	embedder := &testEmbedder{
		embeddings: map[string][]float32{
			"hello": {1, 2, 3},
		},
	}

	got, err := embedder.Embed(
		context.Background(),
		"hello",
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	got[0] = 999

	again, err := embedder.Embed(
		context.Background(),
		"hello",
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if again[0] != 1 {
		t.Fatalf(
			"embedder exposed mutable internal vector",
		)
	}
}
