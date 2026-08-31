package rag

import (
	"errors"
	"math"
	"testing"
)

func TestCosineSimilarity_IdenticalVectors(
	t *testing.T,
) {
	got, err := CosineSimilarity(
		[]float32{1, 2, 3},
		[]float32{1, 2, 3},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if math.Abs(float64(got-1)) > 0.00001 {
		t.Fatalf(
			"expected similarity 1, got %v",
			got,
		)
	}
}

func TestCosineSimilarity_OppositeVectors(
	t *testing.T,
) {
	got, err := CosineSimilarity(
		[]float32{1, 2, 3},
		[]float32{-1, -2, -3},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if math.Abs(float64(got+1)) > 0.00001 {
		t.Fatalf(
			"expected similarity -1, got %v",
			got,
		)
	}
}

func TestCosineSimilarity_OrthogonalVectors(
	t *testing.T,
) {
	got, err := CosineSimilarity(
		[]float32{1, 0},
		[]float32{0, 1},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if math.Abs(float64(got)) > 0.00001 {
		t.Fatalf(
			"expected similarity 0, got %v",
			got,
		)
	}
}

func TestCosineSimilarity_RejectsDifferentDimensions(
	t *testing.T,
) {
	_, err := CosineSimilarity(
		[]float32{1, 2},
		[]float32{1, 2, 3},
	)

	if err == nil {
		t.Fatal("expected dimension mismatch error")
	}
}

func TestCosineSimilarity_RejectsZeroVector(
	t *testing.T,
) {
	_, err := CosineSimilarity(
		[]float32{0, 0},
		[]float32{1, 2},
	)

	if !errors.Is(err, ErrZeroVector) {
		t.Fatalf(
			"expected ErrZeroVector, got %v",
			err,
		)
	}
}

func TestCosineSimilarity_ZeroVectorOnBothSides(
	t *testing.T,
) {
	_, err := CosineSimilarity(
		[]float32{0, 0},
		[]float32{0, 0},
	)

	if !errors.Is(err, ErrZeroVector) {
		t.Fatalf(
			"expected ErrZeroVector, got %v",
			err,
		)
	}
}
