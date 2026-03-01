package openai

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"

	"go.uber.org/zap"
)

const embeddingDimensions = 1536

// FakeEmbedder returns deterministic, normalized 1536-dim vectors derived from
// a SHA-256 hash of each input text. No network calls are made.
// Intended for use in tests only.
type FakeEmbedder struct{}

func (f *FakeEmbedder) GenerateEmbeddings(_ context.Context, _ *zap.Logger, texts []string) (*EmbeddingResult, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = hashToVector(text)
	}
	return &EmbeddingResult{Embeddings: embeddings}, nil
}

// hashToVector derives a deterministic normalized float32 vector from a string.
func hashToVector(text string) []float32 {
	vec := make([]float32, embeddingDimensions)

	// Use repeated SHA-256 rounds seeded by input to fill all dimensions.
	seed := []byte(text)
	for i := 0; i < embeddingDimensions; i += 8 {
		h := sha256.Sum256(seed)
		seed = h[:]
		for j := 0; j < 8 && i+j < embeddingDimensions; j++ {
			bits := binary.LittleEndian.Uint32(h[j*4 : j*4+4])
			// Map uint32 range to [-1, 1]
			vec[i+j] = float32(bits)/float32(math.MaxUint32)*2 - 1
		}
	}

	// L2-normalize
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}

	return vec
}
