package repos

import (
	"context"
	"errors"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/pkg/pgconv"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNotFound        = errors.New("repo not found")
	ErrTokenRequired   = errors.New("git token is required for private repositories")
)

func Create(ctx context.Context, params CreateParams) (*Repo, error) {
	if params.IsPrivate && params.GitTokenID == nil {
		return nil, ErrTokenRequired
	}

	now := time.Now().UnixNano()
	branch := params.Branch
	if branch == "" {
		branch = "main"
	}

	dbRepo, err := db.Tx1(ctx, func(q *db.Queries) (db.Repo, error) {
		return q.CreateRepo(ctx, db.CreateRepoParams{
			WorkspaceID:  params.WorkspaceID,
			GitTokenID:   pgconv.ToInt8(params.GitTokenID),
			Url:          params.URL,
			Branch:       branch,
			IsPrivate:    params.IsPrivate,
			Status:       string(StatusPending),
			Created:      now,
			Updated:      now,
		})
	})
	if err != nil {
		return nil, err
	}

	return toRepo(dbRepo), nil
}

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

func List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 20
	}

	type listData struct {
		repos []db.Repo
		total int64
	}

	data, err := db.Tx1(ctx, func(q *db.Queries) (listData, error) {
		dbRepos, err := q.ListReposByWorkspace(ctx, db.ListReposByWorkspaceParams{
			WorkspaceID: params.WorkspaceID,
			Limit:       int32(params.Limit),
			Offset:      int32(params.Offset),
		})
		if err != nil {
			return listData{}, err
		}

		total, err := q.CountReposByWorkspace(ctx, params.WorkspaceID)
		if err != nil {
			return listData{}, err
		}

		return listData{repos: dbRepos, total: total}, nil
	})
	if err != nil {
		return nil, err
	}

	repos := make([]Repo, len(data.repos))
	for i, r := range data.repos {
		repos[i] = *toRepo(r)
	}

	return &ListResult{Repos: repos, Total: data.total}, nil
}

func UpdateStatus(ctx context.Context, id int64, status Status, errMsg *string) (*Repo, error) {
	now := time.Now().UnixNano()

	var indexedAt *int64
	if status == StatusReady {
		indexedAt = &now
	}

	dbRepo, err := db.Tx1(ctx, func(q *db.Queries) (db.Repo, error) {
		return q.UpdateRepoStatus(ctx, db.UpdateRepoStatusParams{
			ID:           id,
			Status:       string(status),
			IndexedAt:    pgconv.ToInt8(indexedAt),
			ErrorMessage: pgconv.ToText(errMsg),
			Updated:      now,
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

func Update(ctx context.Context, id int64, params UpdateParams) (*Repo, error) {
	if params.IsPrivate && params.GitTokenID == nil {
		return nil, ErrTokenRequired
	}

	now := time.Now().UnixNano()

	dbRepo, err := db.Tx1(ctx, func(q *db.Queries) (db.Repo, error) {
		return q.UpdateRepo(ctx, db.UpdateRepoParams{
			ID:         id,
			GitTokenID: pgconv.ToInt8(params.GitTokenID),
			Url:        params.URL,
			Branch:     params.Branch,
			IsPrivate:  params.IsPrivate,
			Updated:    now,
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

func Delete(ctx context.Context, id int64) error {
	return db.Tx(ctx, func(q *db.Queries) error {
		_, err := q.GetRepoByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		return q.DeleteRepo(ctx, id)
	})
}

// ListPending returns repos with status = pending, ordered by creation time.
func ListPending(ctx context.Context, limit int) ([]Repo, error) {
	dbRepos, err := db.Query1(ctx, func(q *db.Queries) ([]db.Repo, error) {
		return q.ListReposByStatus(ctx, db.ListReposByStatusParams{
			Status: string(StatusPending),
			Limit:  int32(limit),
		})
	})
	if err != nil {
		return nil, err
	}

	repos := make([]Repo, len(dbRepos))
	for i, r := range dbRepos {
		repos[i] = *toRepo(r)
	}
	return repos, nil
}

// CountByGitToken returns how many repos reference the given git token.
func CountByGitToken(ctx context.Context, tokenID int64) (int64, error) {
	return db.Query1(ctx, func(q *db.Queries) (int64, error) {
		return q.CountReposByGitToken(ctx, pgtype.Int8{Int64: tokenID, Valid: true})
	})
}

func toRepo(r db.Repo) *Repo {
	return &Repo{
		ID:           r.ID,
		WorkspaceID:  r.WorkspaceID,
		GitTokenID:   pgconv.FromInt8(r.GitTokenID),
		URL:          r.Url,
		Branch:       r.Branch,
		IsPrivate:    r.IsPrivate,
		Status:       Status(r.Status),
		IndexedAt:    pgconv.FromInt8(r.IndexedAt),
		ErrorMessage: pgconv.FromText(r.ErrorMessage),
		Created:      r.Created,
		Updated:      r.Updated,
	}
}
