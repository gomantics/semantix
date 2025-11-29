package repos

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gomantics/semantix/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	DefaultBranch = "main"
)

var (
	ErrNotFound      = errors.New("repository not found")
	ErrAlreadyExists = errors.New("repository already exists")
	ErrAlreadyActive = errors.New("repository is already being indexed")
)

// Create creates a new repository or returns existing one
func Create(ctx context.Context, params CreateParams) (*Repo, error) {
	url := normalizeURL(params.URL)
	owner, name := parseRepoURL(url)

	// Check if already exists in this workspace
	existing, err := GetByWorkspaceAndURL(ctx, params.WorkspaceID, url)
	if err == nil {
		return existing, ErrAlreadyExists
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	now := time.Now().Unix()

	var gitTokenID pgtype.Int8
	if params.GitTokenID != nil {
		gitTokenID = pgtype.Int8{Int64: *params.GitTokenID, Valid: true}
	}

	dbRepo, err := db.Query1(ctx, func(q *db.Queries) (db.Repo, error) {
		return q.CreateRepo(ctx, db.CreateRepoParams{
			WorkspaceID:   params.WorkspaceID,
			GitTokenID:    gitTokenID,
			Url:           url,
			Name:          name,
			Owner:         owner,
			DefaultBranch: DefaultBranch,
			Status:        StatusPending.String(),
			Created:       now,
			Updated:       now,
		})
	})
	if err != nil {
		return nil, err
	}

	return toRepo(dbRepo), nil
}

// GetByID retrieves a repository by ID
func GetByID(ctx context.Context, id int64) (*Repo, error) {
	dbRepo, err := db.Query1(ctx, func(q *db.Queries) (db.Repo, error) {
		return q.GetRepoByID(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toRepo(dbRepo), nil
}

// GetByWorkspaceAndURL retrieves a repository by workspace ID and URL
func GetByWorkspaceAndURL(ctx context.Context, workspaceID int64, url string) (*Repo, error) {
	dbRepo, err := db.Query1(ctx, func(q *db.Queries) (db.Repo, error) {
		return q.GetRepoByWorkspaceAndURL(ctx, db.GetRepoByWorkspaceAndURLParams{
			WorkspaceID: workspaceID,
			Url:         normalizeURL(url),
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toRepo(dbRepo), nil
}

// GetByIDWithStats retrieves a repository with file and chunk counts
func GetByIDWithStats(ctx context.Context, id int64) (*RepoWithStats, error) {
	var dbRepo db.Repo
	var fileCount, chunkCount int64

	err := db.Query(ctx, func(q *db.Queries) error {
		var err error
		dbRepo, err = q.GetRepoByID(ctx, id)
		if err != nil {
			return err
		}

		fileCount, _ = q.CountFilesByRepoID(ctx, id)
		chunkCount, _ = q.SumChunksByRepoID(ctx, id)
		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	repo := toRepo(dbRepo)
	return &RepoWithStats{
		Repo:       *repo,
		FileCount:  fileCount,
		ChunkCount: chunkCount,
	}, nil
}

// List retrieves repositories with optional filtering
func List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 20
	}

	var dbRepos []db.Repo
	var total int64

	err := db.Query(ctx, func(q *db.Queries) error {
		var err error
		if params.Status != nil {
			dbRepos, err = q.ListReposByWorkspaceAndStatus(ctx, db.ListReposByWorkspaceAndStatusParams{
				WorkspaceID: params.WorkspaceID,
				Status:      params.Status.String(),
				Limit:       int32(params.Limit),
				Offset:      int32(params.Offset),
			})
			if err != nil {
				return err
			}
			total, err = q.CountReposByWorkspaceAndStatus(ctx, db.CountReposByWorkspaceAndStatusParams{
				WorkspaceID: params.WorkspaceID,
				Status:      params.Status.String(),
			})
			return err
		}

		dbRepos, err = q.ListReposByWorkspace(ctx, db.ListReposByWorkspaceParams{
			WorkspaceID: params.WorkspaceID,
			Limit:       int32(params.Limit),
			Offset:      int32(params.Offset),
		})
		if err != nil {
			return err
		}
		total, err = q.CountReposByWorkspace(ctx, params.WorkspaceID)
		return err
	})
	if err != nil {
		return nil, err
	}

	repos := make([]Repo, len(dbRepos))
	for i, dbRepo := range dbRepos {
		repos[i] = *toRepo(dbRepo)
	}

	return &ListResult{Repos: repos, Total: total}, nil
}

// Delete removes a repository by ID
func Delete(ctx context.Context, id int64) error {
	// First check it exists
	_, err := GetByID(ctx, id)
	if err != nil {
		return err
	}

	return db.Query(ctx, func(q *db.Queries) error {
		return q.DeleteRepo(ctx, id)
	})
}

// UpdateStatus updates the status of a repository
func UpdateStatus(ctx context.Context, id int64, status Status) error {
	now := time.Now().Unix()
	return db.Query(ctx, func(q *db.Queries) error {
		_, err := q.UpdateRepoStatus(ctx, db.UpdateRepoStatusParams{
			ID:      id,
			Status:  status.String(),
			Updated: now,
		})
		return err
	})
}

// SetError sets the repository status to failed with an error message
func SetError(ctx context.Context, id int64, errMsg string) error {
	now := time.Now().Unix()
	return db.Query(ctx, func(q *db.Queries) error {
		_, err := q.UpdateRepoStatus(ctx, db.UpdateRepoStatusParams{
			ID:      id,
			Status:  StatusFailed.String(),
			Error:   pgtype.Text{String: errMsg, Valid: true},
			Updated: now,
		})
		return err
	})
}

// UpdateAfterClone updates repo with commit SHA and sets status to indexing
func UpdateAfterClone(ctx context.Context, id int64, commitSHA string) error {
	now := time.Now().Unix()
	return db.Query(ctx, func(q *db.Queries) error {
		_, err := q.UpdateRepoAfterClone(ctx, db.UpdateRepoAfterCloneParams{
			ID:            id,
			LastCommitSha: pgtype.Text{String: commitSHA, Valid: true},
			Status:        StatusIndexing.String(),
			Updated:       now,
		})
		return err
	})
}

// UpdateGitToken updates the git token for a repository
func UpdateGitToken(ctx context.Context, id int64, gitTokenID *int64) error {
	now := time.Now().Unix()
	var tokenID pgtype.Int8
	if gitTokenID != nil {
		tokenID = pgtype.Int8{Int64: *gitTokenID, Valid: true}
	}
	return db.Query(ctx, func(q *db.Queries) error {
		_, err := q.UpdateRepoGitToken(ctx, db.UpdateRepoGitTokenParams{
			ID:         id,
			GitTokenID: tokenID,
			Updated:    now,
		})
		return err
	})
}

// TriggerReindex queues a repository for re-indexing
func TriggerReindex(ctx context.Context, id int64) error {
	repo, err := GetByID(ctx, id)
	if err != nil {
		return err
	}

	if repo.Status.IsActive() {
		return ErrAlreadyActive
	}

	return UpdateStatus(ctx, id, StatusPending)
}

// ClaimPending atomically claims a pending repository for processing
func ClaimPending(ctx context.Context) (*Repo, error) {
	now := time.Now().Unix()
	dbRepo, err := db.Query1(ctx, func(q *db.Queries) (db.Repo, error) {
		return q.ClaimPendingRepo(ctx, now)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toRepo(dbRepo), nil
}

// toRepo converts a db.Repo to domain Repo
func toRepo(dbRepo db.Repo) *Repo {
	var gitTokenID *int64
	if dbRepo.GitTokenID.Valid {
		gitTokenID = &dbRepo.GitTokenID.Int64
	}

	return &Repo{
		ID:            dbRepo.ID,
		WorkspaceID:   dbRepo.WorkspaceID,
		GitTokenID:    gitTokenID,
		URL:           dbRepo.Url,
		Name:          dbRepo.Name,
		Owner:         dbRepo.Owner,
		DefaultBranch: dbRepo.DefaultBranch,
		LastCommitSHA: dbRepo.LastCommitSha.String,
		Status:        Status(dbRepo.Status),
		Error:         dbRepo.Error.String,
		Created:       dbRepo.Created,
		Updated:       dbRepo.Updated,
	}
}

// normalizeURL normalizes a repository URL
func normalizeURL(url string) string {
	// Remove .git suffix
	if strings.HasSuffix(url, ".git") {
		url = url[:len(url)-4]
	}
	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")
	return url
}

// parseRepoURL extracts owner and name from a repository URL
func parseRepoURL(url string) (owner, name string) {
	url = normalizeURL(url)

	// Split by / or :
	var parts []string
	var current string
	for _, c := range url {
		if c == '/' || c == ':' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) >= 2 {
		owner = parts[len(parts)-2]
		name = parts[len(parts)-1]
	}
	return owner, name
}
