package search

import (
	"net/http"

	"github.com/gomantics/semantix/internal/api/web"
)

type SearchRequest struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
	RepoID *int64 `json:"repo_id,omitempty"`
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

// TODO: implement semantic search via Qdrant
func Search(c web.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "search not yet implemented",
	})
}
