package openai

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

const (
	embeddingModel = openai.SmallEmbedding3
	batchSize      = 2048
	maxRetries     = 5
	baseDelay      = 500 * time.Millisecond
)

// EmbeddingResult holds a batch of embeddings mapped by their original index.
type EmbeddingResult struct {
	Embeddings [][]float32
}

// Embedder is the interface for generating text embeddings.
type Embedder interface {
	GenerateEmbeddings(ctx context.Context, l *zap.Logger, texts []string) (*EmbeddingResult, error)
}

// Client wraps the OpenAI SDK client and implements Embedder.
type Client struct {
	inner *openai.Client
}

// NewClient creates a Client from an API key.
func NewClient(apiKey string) *Client {
	return &Client{inner: openai.NewClient(apiKey)}
}

func (c *Client) GenerateEmbeddings(ctx context.Context, l *zap.Logger, texts []string) (*EmbeddingResult, error) {
	if len(texts) == 0 {
		return &EmbeddingResult{}, nil
	}

	allEmbeddings := make([][]float32, len(texts))

	for batchStart := 0; batchStart < len(texts); batchStart += batchSize {
		batchEnd := min(batchStart+batchSize, len(texts))
		batch := texts[batchStart:batchEnd]

		resp, err := createEmbeddingsWithRetry(ctx, l, c.inner, batch)
		if err != nil {
			return nil, fmt.Errorf("embedding batch %d-%d: %w", batchStart, batchEnd, err)
		}

		for _, emb := range resp.Data {
			allEmbeddings[batchStart+emb.Index] = emb.Embedding
		}
	}

	return &EmbeddingResult{Embeddings: allEmbeddings}, nil
}

func createEmbeddingsWithRetry(ctx context.Context, l *zap.Logger, client *openai.Client, texts []string) (openai.EmbeddingResponse, error) {
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			delay := min(time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1))), 30 * time.Second)

			l.Warn("retrying embedding request",
				zap.Int("attempt", attempt+1),
				zap.Duration("delay", delay),
				zap.Error(lastErr),
			)

			select {
			case <-ctx.Done():
				return openai.EmbeddingResponse{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
			Input: texts,
			Model: embeddingModel,
		})
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	return openai.EmbeddingResponse{}, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
