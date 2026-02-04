package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gomantics/semantix/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNotFound      = errors.New("workspace not found")
	ErrAlreadyExists = errors.New("workspace with this slug already exists")
)

// Create creates a new workspace
func Create(ctx context.Context, params CreateParams) (*Workspace, error) {
	// Check if slug already exists
	_, err := GetBySlug(ctx, params.Slug)
	if err == nil {
		return nil, ErrAlreadyExists
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	now := time.Now().UnixNano()
	settings := params.Settings
	if settings == nil {
		settings = make(map[string]any)
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	dbWorkspace, err := db.Query1(ctx, func(q *db.Queries) (db.Workspace, error) {
		return q.CreateWorkspace(ctx, db.CreateWorkspaceParams{
			Name:        params.Name,
			Slug:        params.Slug,
			Description: toPgText(params.Description),
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

// GetByID retrieves a workspace by ID
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

// GetBySlug retrieves a workspace by slug
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

	var dbWorkspaces []db.Workspace
	var total int64

	err := db.Query(ctx, func(q *db.Queries) error {
		var err error
		dbWorkspaces, err = q.ListWorkspaces(ctx, db.ListWorkspacesParams{
			Limit:  int32(params.Limit),
			Offset: int32(params.Offset),
		})
		if err != nil {
			return err
		}
		total, err = q.CountWorkspaces(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	workspaces := make([]Workspace, len(dbWorkspaces))
	for i, dbWs := range dbWorkspaces {
		workspaces[i] = *toWorkspace(dbWs)
	}

	return &ListResult{Workspaces: workspaces, Total: total}, nil
}

// Update updates a workspace
func Update(ctx context.Context, id int64, params UpdateParams) (*Workspace, error) {
	// Check if new slug conflicts with another workspace
	existing, err := GetBySlug(ctx, params.Slug)
	if err == nil && existing.ID != id {
		return nil, ErrAlreadyExists
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	now := time.Now().UnixNano()
	settings := params.Settings
	if settings == nil {
		settings = make(map[string]any)
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	dbWorkspace, err := db.Query1(ctx, func(q *db.Queries) (db.Workspace, error) {
		return q.UpdateWorkspace(ctx, db.UpdateWorkspaceParams{
			ID:          id,
			Name:        params.Name,
			Slug:        params.Slug,
			Description: toPgText(params.Description),
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

// Delete removes a workspace by ID
func Delete(ctx context.Context, id int64) error {
	_, err := GetByID(ctx, id)
	if err != nil {
		return err
	}

	return db.Query(ctx, func(q *db.Queries) error {
		return q.DeleteWorkspace(ctx, id)
	})
}

func toWorkspace(dbWorkspace db.Workspace) *Workspace {
	var description *string
	if dbWorkspace.Description.Valid {
		description = &dbWorkspace.Description.String
	}

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
		Description: description,
		Settings:    settings,
		Created:     dbWorkspace.Created,
		Updated:     dbWorkspace.Updated,
	}
}

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
