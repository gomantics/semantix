package openai

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

const (
	embeddingModel      = openai.SmallEmbedding3
	embeddingModelTikID = "text-embedding-3-small"
	batchSize           = 2048
	maxRetries          = 5
	baseDelay           = 500 * time.Millisecond
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

	estimatedTokens := countTokens(l, texts)
	l.Info("embedding request",
		zap.Int("texts", len(texts)),
		zap.Int("estimated_tokens", estimatedTokens),
		zap.String("model", embeddingModelTikID),
	)

	allEmbeddings := make([][]float32, len(texts))
	var totalPromptTokens int

	for batchStart := 0; batchStart < len(texts); batchStart += batchSize {
		batchEnd := min(batchStart+batchSize, len(texts))
		batch := texts[batchStart:batchEnd]

		resp, err := createEmbeddingsWithRetry(ctx, l, c.inner, batch)
		if err != nil {
			return nil, fmt.Errorf("embedding batch %d-%d: %w", batchStart, batchEnd, err)
		}

		totalPromptTokens += resp.Usage.PromptTokens

		for _, emb := range resp.Data {
			allEmbeddings[batchStart+emb.Index] = emb.Embedding
		}
	}

	l.Info("embedding complete",
		zap.Int("texts", len(texts)),
		zap.Int("prompt_tokens_billed", totalPromptTokens),
	)

	return &EmbeddingResult{Embeddings: allEmbeddings}, nil
}

// countTokens estimates the total token count for texts using tiktoken.
// Falls back to a character-based approximation if the encoder is unavailable.
func countTokens(l *zap.Logger, texts []string) int {
	enc, err := tiktoken.EncodingForModel(embeddingModelTikID)
	if err != nil {
		l.Warn("tiktoken encoder unavailable, using char approximation", zap.Error(err))
		total := 0
		for _, t := range texts {
			total += len(t) / 4
		}
		return total
	}

	total := 0
	for _, t := range texts {
		total += len(enc.Encode(t, nil, nil))
	}
	return total
}

func createEmbeddingsWithRetry(ctx context.Context, l *zap.Logger, client *openai.Client, texts []string) (openai.EmbeddingResponse, error) {
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			delay := min(time.Duration(float64(baseDelay)*math.Pow(2, float64(attempt-1))), 30*time.Second)

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

		// Do not retry on permanent errors - auth failures, bad requests, or
		// model-not-found will never succeed regardless of how many retries we do.
		if apiErr, ok := err.(*openai.APIError); ok {
			switch apiErr.HTTPStatusCode {
			case http.StatusUnauthorized, http.StatusForbidden,
				http.StatusBadRequest, http.StatusNotFound:
				return openai.EmbeddingResponse{}, fmt.Errorf("permanent error (status %d), not retrying: %w", apiErr.HTTPStatusCode, err)
			}
		}
	}

	return openai.EmbeddingResponse{}, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
