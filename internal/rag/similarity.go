package rag

import (
	"errors"
	"math"
)

var ErrZeroVector = errors.New("cannot calculate similarity for zero vector")

func CosineSimilarity(
	a []float32,
	b []float32,
) (float32, error) {

	if len(a) != len(b) {
		return 0, errors.New("vectors must have equal dimensions")
	}

	if len(a) == 0 {
		return 0, ErrZeroVector
	}

	var dot float64
	var normA float64
	var normB float64

	for i := range a {
		x := float64(a[i])
		y := float64(b[i])

		dot += x * y
		normA += x * x
		normB += y * y
	}

	if normA == 0 || normB == 0 {
		return 0, ErrZeroVector
	}

	return float32(
		dot / (math.Sqrt(normA) * math.Sqrt(normB)),
	), nil
}
