package rag

import (
	"errors"
	"sort"
)

var (
	ErrEmptyDocumentID = errors.New("document id cannot be empty")
	ErrEmptyVector     = errors.New("document vector cannot be empty")
)

type Document struct {
	ID       string
	Content  string
	Vector   []float32
	Metadata DocumentMetadata
}

type DocumentMetadata struct {
	SourceDocumentID string
	ChunkIndex       int
}

type SearchResult struct {
	Document   Document
	Similarity float32
}

type InMemoryVectorStore struct {
	documents []Document
}

func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		documents: make([]Document, 0),
	}
}

func (s *InMemoryVectorStore) Add(
	document Document,
) error {

	if document.ID == "" {
		return ErrEmptyDocumentID
	}

	if len(document.Vector) == 0 {
		return ErrEmptyVector
	}

	// Copy the vector so callers cannot mutate
	// the stored representation accidentally.
	document.Vector = append(
		[]float32(nil),
		document.Vector...,
	)

	s.documents = append(
		s.documents,
		document,
	)

	return nil
}

func (s *InMemoryVectorStore) Search(
	query []float32,
	topK int,
) []SearchResult {

	if len(query) == 0 || topK <= 0 {
		return nil
	}

	results := make(
		[]SearchResult,
		0,
		len(s.documents),
	)

	for _, document := range s.documents {

		similarity, err := CosineSimilarity(
			query,
			document.Vector,
		)

		if err != nil {
			// A vector with incompatible dimensions
			// cannot participate in this search.
			continue
		}

		results = append(
			results,
			SearchResult{
				Document:   document,
				Similarity: similarity,
			},
		)
	}

	sort.SliceStable(
		results,
		func(i, j int) bool {
			return results[i].Similarity >
				results[j].Similarity
		},
	)

	if topK > len(results) {
		topK = len(results)
	}

	return results[:topK]
}
