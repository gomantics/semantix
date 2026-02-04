package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gomantics/semantix/db"
	"github.com/gomantics/semantix/pkg/pgconv"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound      = errors.New("workspace not found")
	ErrAlreadyExists = errors.New("workspace with this slug already exists")
)

func Create(ctx context.Context, params CreateParams) (*Workspace, error) {
	now := time.Now().UnixNano()
	settings := params.Settings
	if settings == nil {
		settings = make(map[string]any)
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	dbWorkspace, err := db.Tx1(ctx, func(q *db.Queries) (db.Workspace, error) {
		_, err := q.GetWorkspaceBySlug(ctx, params.Slug)
		if err == nil {
			return db.Workspace{}, ErrAlreadyExists
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return db.Workspace{}, err
		}

		return q.CreateWorkspace(ctx, db.CreateWorkspaceParams{
			Name:        params.Name,
			Slug:        params.Slug,
			Description: pgconv.ToText(params.Description),
			Settings:    settingsJSON,
			Created:     now,
			Updated:     now,
		})
	})
	if err != nil {
		return nil, err
	}

	return toWorkspace(dbWorkspace), nil
}

func GetByID(ctx context.Context, id int64) (*Workspace, error) {
	dbWorkspace, err := db.Query1(ctx, func(q *db.Queries) (db.Workspace, error) {
		return q.GetWorkspaceByID(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toWorkspace(dbWorkspace), nil
}

func GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	dbWorkspace, err := db.Query1(ctx, func(q *db.Queries) (db.Workspace, error) {
		return q.GetWorkspaceBySlug(ctx, slug)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toWorkspace(dbWorkspace), nil
}

// List retrieves workspaces with pagination
func List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 20
	}

	type listData struct {
		workspaces []db.Workspace
		total      int64
	}

	data, err := db.Tx1(ctx, func(q *db.Queries) (listData, error) {
		dbWorkspaces, err := q.ListWorkspaces(ctx, db.ListWorkspacesParams{
			Limit:  int32(params.Limit),
			Offset: int32(params.Offset),
		})
		if err != nil {
			return listData{}, err
		}

		total, err := q.CountWorkspaces(ctx)
		if err != nil {
			return listData{}, err
		}

		return listData{workspaces: dbWorkspaces, total: total}, nil
	})
	if err != nil {
		return nil, err
	}

	workspaces := make([]Workspace, len(data.workspaces))
	for i, dbWs := range data.workspaces {
		workspaces[i] = *toWorkspace(dbWs)
	}

	return &ListResult{Workspaces: workspaces, Total: data.total}, nil
}

func Update(ctx context.Context, id int64, params UpdateParams) (*Workspace, error) {
	now := time.Now().UnixNano()
	settings := params.Settings
	if settings == nil {
		settings = make(map[string]any)
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	dbWorkspace, err := db.Tx1(ctx, func(q *db.Queries) (db.Workspace, error) {
		existing, err := q.GetWorkspaceBySlug(ctx, params.Slug)
		if err == nil && existing.ID != id {
			return db.Workspace{}, ErrAlreadyExists
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return db.Workspace{}, err
		}

		return q.UpdateWorkspace(ctx, db.UpdateWorkspaceParams{
			ID:          id,
			Name:        params.Name,
			Slug:        params.Slug,
			Description: pgconv.ToText(params.Description),
			Settings:    settingsJSON,
			Updated:     now,
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return toWorkspace(dbWorkspace), nil
}

func Delete(ctx context.Context, id int64) error {
	return db.Tx(ctx, func(q *db.Queries) error {
		_, err := q.GetWorkspaceByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		return q.DeleteWorkspace(ctx, id)
	})
}

func toWorkspace(dbWorkspace db.Workspace) *Workspace {
	var settings map[string]any
	if len(dbWorkspace.Settings) > 0 {
		_ = json.Unmarshal(dbWorkspace.Settings, &settings)
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	return &Workspace{
		ID:          dbWorkspace.ID,
		Name:        dbWorkspace.Name,
		Slug:        dbWorkspace.Slug,
		Description: pgconv.FromText(dbWorkspace.Description),
		Settings:    settings,
		Created:     dbWorkspace.Created,
		Updated:     dbWorkspace.Updated,
	}
}
