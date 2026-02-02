package gptcache

import (
	"math"
	"strings"
)

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value between -1 and 1.
func CosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) || len(vec1) == 0 {
		return 0
	}

	var dot, norm1, norm2 float64
	for i := range vec1 {
		dot += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	norm1 = math.Sqrt(norm1)
	norm2 = math.Sqrt(norm2)

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dot / (norm1 * norm2)
}

// NormalizeL2 performs L2 normalization on a vector.
func NormalizeL2(vec []float64) []float64 {
	if len(vec) == 0 {
		return vec
	}

	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return vec
	}

	result := make([]float64, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}

// EmbeddingMatcher implements SemanticMatcher using embedding vectors.
type EmbeddingMatcher struct {
	// EmbedFunc converts a query string to an embedding vector.
	EmbedFunc func(query string) ([]float64, error)
}

// Similarity computes cosine similarity between embeddings of two queries.
func (m *EmbeddingMatcher) Similarity(query1, query2 string) (float64, error) {
	if m.EmbedFunc == nil {
		return exactMatchSimilarity(query1, query2), nil
	}

	emb1, err := m.EmbedFunc(query1)
	if err != nil {
		return 0, err
	}

	emb2, err := m.EmbedFunc(query2)
	if err != nil {
		return 0, err
	}

	sim := CosineSimilarity(emb1, emb2)
	// Normalize from [-1, 1] to [0, 1]
	return (sim + 1) / 2, nil
}

// exactMatchSimilarity returns 1.0 for exact matches, 0.0 otherwise.
func exactMatchSimilarity(a, b string) float64 {
	if strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) {
		return 1.0
	}
	return 0.0
}
