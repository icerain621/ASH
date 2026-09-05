package rag

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

// Embedder turns text batches into dense vectors.
type Embedder interface {
	Embed(texts []string) ([][]float32, error)
	Dim() int
}

// HashEmbedder produces deterministic vectors from text hashes (no network).
type HashEmbedder struct {
	dim int
}

func NewHashEmbedder(dim int) *HashEmbedder {
	if dim <= 0 {
		dim = 64
	}
	return &HashEmbedder{dim: dim}
}

func (e *HashEmbedder) Dim() int {
	if e == nil {
		return 0
	}
	return e.dim
}

func (e *HashEmbedder) Embed(texts []string) ([][]float32, error) {
	if e == nil {
		return nil, errors.New("embedder is nil")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		if text == "" {
			return nil, errors.New("empty text")
		}
		out[i] = hashToVector(text, e.dim)
	}
	return out, nil
}

func hashToVector(text string, dim int) []float32 {
	sum := sha256.Sum256([]byte(text))
	vec := make([]float32, dim)
	for i := 0; i < dim; i++ {
		vec[i] = float32(sum[i%len(sum)]) / 255.0
	}
	return vec
}

// Ensure HashEmbedder implements Embedder.
var _ Embedder = (*HashEmbedder)(nil)

// DefaultHashEmbedderDim is the POC vector size for HashEmbedder.
const DefaultHashEmbedderDim = 64

func DefaultHashEmbedder() *HashEmbedder {
	return NewHashEmbedder(DefaultHashEmbedderDim)
}

func (e *HashEmbedder) String() string {
	return fmt.Sprintf("HashEmbedder(dim=%d)", e.dim)
}
