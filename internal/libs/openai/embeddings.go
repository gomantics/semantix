package openai

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gomantics/semantix/config"
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

var defaultEmbedder Embedder

// Init initializes the OpenAI client from config and sets the default embedder.
func Init() error {
	apiKey := config.Openai.ApiKey()
	if apiKey != "" {
		defaultEmbedder = &Client{inner: openai.NewClient(apiKey)}
	}
	return nil
}

// SetDefaultEmbedder replaces the default embedder. Intended for use in tests only.
func SetDefaultEmbedder(e Embedder) {
	defaultEmbedder = e
}

// GetDefaultEmbedder returns the current default embedder.
func GetDefaultEmbedder() Embedder {
	return defaultEmbedder
}

// GenerateEmbeddings creates embeddings using the default embedder.
func GenerateEmbeddings(ctx context.Context, l *zap.Logger, texts []string) (*EmbeddingResult, error) {
	if defaultEmbedder == nil {
		return nil, fmt.Errorf("openai client not initialized: set CONFIG_OPENAI_API_KEY")
	}
	return defaultEmbedder.GenerateEmbeddings(ctx, l, texts)
}

// Client wraps the OpenAI SDK client and implements Embedder.
type Client struct {
	inner *openai.Client
}

// GetClient returns the underlying OpenAI SDK client.
func (c *Client) GetClient() *openai.Client {
	return c.inner
}

func (c *Client) GenerateEmbeddings(ctx context.Context, l *zap.Logger, texts []string) (*EmbeddingResult, error) {
	if len(texts) == 0 {
		return &EmbeddingResult{}, nil
	}

	allEmbeddings := make([][]float32, len(texts))

	for batchStart := 0; batchStart < len(texts); batchStart += batchSize {
		batchEnd := min(batchStart + batchSize, len(texts))
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
			delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}

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
