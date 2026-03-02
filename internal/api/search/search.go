package search

import (
	"net/http"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/settings"
	"github.com/gomantics/semantix/internal/libs/openai"
	"github.com/gomantics/semantix/internal/qdrant"
	pb "github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

type SearchRequest struct {
	Query   string  `json:"query"`
	Limit   int     `json:"limit,omitempty"`
	RepoIDs []int64 `json:"repo_ids,omitempty"`
}

type SearchResult struct {
	RepoID    int64   `json:"repo_id"`
	FilePath  string  `json:"file_path"`
	Content   string  `json:"content"`
	Language  string  `json:"language,omitempty"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

func Search(c web.Context) error {
	wid, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	var req SearchRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Query == "" {
		return c.BadRequest("query is required")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}

	apiKey, err := settings.GetOpenAIKey()
	if err != nil {
		c.L.Error("failed to load openai key", zap.Error(err))
		return c.InternalError("openai not configured")
	}

	ctx := c.Request().Context()

	embedder := openai.NewClient(apiKey)
	result, err := embedder.GenerateEmbeddings(ctx, c.L, []string{req.Query})
	if err != nil {
		c.L.Error("failed to generate query embedding", zap.Error(err))
		return c.InternalError("failed to generate embedding")
	}
	if len(result.Embeddings) == 0 {
		return c.InternalError("empty embedding result")
	}
	vector := result.Embeddings[0]

	mustConditions := []*pb.Condition{
		pb.NewMatchInt("workspace_id", wid),
	}

	if len(req.RepoIDs) > 0 {
		mustConditions = append(mustConditions, pb.NewMatchInts("repo_id", req.RepoIDs...))
	}

	filter := &pb.Filter{
		Must: mustConditions,
	}

	points, err := qdrant.SearchPoints(ctx, vector, filter, uint64(limit))
	if err != nil {
		c.L.Error("qdrant search failed", zap.Error(err))
		return c.InternalError("search failed")
	}

	results := make([]SearchResult, 0, len(points))
	for _, p := range points {
		payload := p.GetPayload()
		results = append(results, SearchResult{
			RepoID:    payload["repo_id"].GetIntegerValue(),
			FilePath:  payload["file_path"].GetStringValue(),
			Content:   payload["chunk_content"].GetStringValue(),
			Language:  payload["language"].GetStringValue(),
			StartLine: int(payload["start_line"].GetIntegerValue()),
			EndLine:   int(payload["end_line"].GetIntegerValue()),
			Score:     p.GetScore(),
		})
	}

	return c.JSON(http.StatusOK, SearchResponse{Results: results})
}
