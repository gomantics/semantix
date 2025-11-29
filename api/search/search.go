package search

import (
	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/libs/milvus"
	"github.com/gomantics/semantix/libs/openai"
	"go.uber.org/zap"
)

// SearchRequest is the request body for searching
type SearchRequest struct {
	Query       string `json:"query"`
	WorkspaceID int64  `json:"workspace_id"`      // Required for workspace isolation
	RepoID      int64  `json:"repo_id,omitempty"` // Optional filter within workspace
	Limit       int    `json:"limit,omitempty"`
}

// SearchResult is a single search result
type SearchResult struct {
	FilePath   string  `json:"file_path"`
	Content    string  `json:"content"`
	StartLine  int64   `json:"start_line"`
	EndLine    int64   `json:"end_line"`
	Similarity float32 `json:"similarity"`
	Language   string  `json:"language"`
	RepoID     int64   `json:"repo_id"`
}

// SearchResponse is the response for searching
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// Search handles POST /v1/search
func Search(c web.Context) error {
	ctx := c.Request().Context()

	var req SearchRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Query == "" {
		return c.BadRequest("query is required")
	}

	if req.WorkspaceID <= 0 {
		return c.BadRequest("workspace_id is required")
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}

	// Generate embedding for the query
	embedding, err := openai.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		c.L.Error("failed to generate embedding", zap.Error(err))
		return c.InternalError("failed to process query")
	}

	// Search in Milvus (scoped by workspace)
	results, err := milvus.SearchSimilar(ctx, embedding, req.WorkspaceID, req.RepoID, req.Limit)
	if err != nil {
		c.L.Error("failed to search", zap.Error(err))
		return c.InternalError("failed to search")
	}

	// Convert to response format
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		// Convert L2 distance to similarity (lower distance = higher similarity)
		// L2 distance ranges from 0 to infinity, we normalize to 0-1
		similarity := 1.0 / (1.0 + r.Score)

		searchResults[i] = SearchResult{
			FilePath:   r.FilePath,
			Content:    r.Content,
			StartLine:  r.StartLine,
			EndLine:    r.EndLine,
			Similarity: similarity,
			Language:   r.Language,
			RepoID:     r.RepoID,
		}
	}

	c.L.Debug("search completed",
		zap.String("query", req.Query),
		zap.Int64("workspace_id", req.WorkspaceID),
		zap.Int("results", len(searchResults)),
	)

	return c.OK(SearchResponse{
		Results: searchResults,
	})
}
