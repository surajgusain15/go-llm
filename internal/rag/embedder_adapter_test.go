package rag

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type testLLMEmbedder struct {
	text      string
	embedding []float32
	err       error
}

func (e *testLLMEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	e.text = text

	if e.err != nil {
		return nil, e.err
	}

	return e.embedding, nil
}

func TestEmbedderAdapter_ForwardsText(t *testing.T) {
	provider := &testLLMEmbedder{
		embedding: []float32{0.1, 0.2, 0.3},
	}

	adapter := NewEmbedderAdapter(provider)

	got, err := adapter.Embed(
		context.Background(),
		"database timeout",
	)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if provider.text != "database timeout" {
		t.Fatalf(
			"provider text = %q, want %q",
			provider.text,
			"database timeout",
		)
	}

	want := []float32{0.1, 0.2, 0.3}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"embedding = %v, want %v",
			got,
			want,
		)
	}
}

func TestEmbedderAdapter_PropagatesError(t *testing.T) {
	expectedErr := errors.New("embedding provider unavailable")

	provider := &testLLMEmbedder{
		err: expectedErr,
	}

	adapter := NewEmbedderAdapter(provider)

	_, err := adapter.Embed(
		context.Background(),
		"database timeout",
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			expectedErr,
		)
	}
}
