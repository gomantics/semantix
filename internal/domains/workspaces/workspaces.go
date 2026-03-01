package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/pkg/pgconv"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("workspace not found")

func Create(ctx context.Context, params CreateParams) (*Workspace, error) {
	now := time.Now().UnixNano()
	settings := params.Settings
	if settings == nil {
		settings = &WorkspaceSettings{}
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	dbWorkspace, err := db.Query1(ctx, func(q *db.Queries) (db.Workspace, error) {
		return q.CreateWorkspace(ctx, db.CreateWorkspaceParams{
			Name:        params.Name,
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
		settings = &WorkspaceSettings{}
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	dbWorkspace, err := db.Query1(ctx, func(q *db.Queries) (db.Workspace, error) {
		return q.UpdateWorkspace(ctx, db.UpdateWorkspaceParams{
			ID:          id,
			Name:        params.Name,
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
	var settings WorkspaceSettings
	if len(dbWorkspace.Settings) > 0 {
		_ = json.Unmarshal(dbWorkspace.Settings, &settings)
	}

	return &Workspace{
		ID:          dbWorkspace.ID,
		Name:        dbWorkspace.Name,
		Description: pgconv.FromText(dbWorkspace.Description),
		Settings:    settings,
		Created:     dbWorkspace.Created,
		Updated:     dbWorkspace.Updated,
	}
}
