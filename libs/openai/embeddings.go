package openai

import (
	"context"
	"fmt"

	"github.com/gomantics/semantix/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

const (
	// EmbeddingModel is the model used for generating embeddings
	EmbeddingModel = "text-embedding-3-small"

	// MaxTokensPerRequest is the maximum tokens per embedding request
	MaxTokensPerRequest = 8000

	// MaxTextsPerBatch is the maximum number of texts per batch request
	MaxTextsPerBatch = 2048
)

// GenerateEmbedding generates an embedding for a single text
func GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	apiKey := config.Openai.ApiKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not configured")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(EmbeddingModel),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.NewOpt(text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	// Convert float64 to float32 for Milvus
	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
func GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	apiKey := config.Openai.ApiKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not configured")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	// Process in batches if needed
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += MaxTextsPerBatch {
		end := i + MaxTextsPerBatch
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := generateBatchEmbeddingsInternal(ctx, client, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to generate batch embeddings (batch %d-%d): %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// generateBatchEmbeddingsInternal generates embeddings for a batch of texts
func generateBatchEmbeddingsInternal(ctx context.Context, client openai.Client, texts []string) ([][]float32, error) {
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(EmbeddingModel),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Data))
	}

	// Convert float64 to float32 for Milvus
	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embedding := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding[j] = float32(v)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// EstimateTokens provides a rough estimate of token count for a text
// This is a simple heuristic: ~4 characters per token for English text
func EstimateTokens(text string) int {
	return len(text) / 4
}

// ChunkTextsForEmbedding splits texts into chunks that fit within token limits
func ChunkTextsForEmbedding(texts []string) [][]string {
	var chunks [][]string
	var currentChunk []string
	currentTokens := 0

	for _, text := range texts {
		tokens := EstimateTokens(text)

		// If a single text exceeds the limit, truncate it
		if tokens > MaxTokensPerRequest {
			// Truncate to fit
			maxChars := MaxTokensPerRequest * 4
			if len(text) > maxChars {
				text = text[:maxChars]
			}
			tokens = EstimateTokens(text)
		}

		// Check if adding this text would exceed limits
		if currentTokens+tokens > MaxTokensPerRequest || len(currentChunk) >= MaxTextsPerBatch {
			if len(currentChunk) > 0 {
				chunks = append(chunks, currentChunk)
				currentChunk = nil
				currentTokens = 0
			}
		}

		currentChunk = append(currentChunk, text)
		currentTokens += tokens
	}

	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
