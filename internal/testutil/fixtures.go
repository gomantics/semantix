package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/gomantics/semantix/internal/db"
)

// Fixtures holds seed data created before tests run.
type Fixtures struct {
	Workspace db.Workspace
}

// LoadFixtures creates a standard workspace used as the base for tests.
func LoadFixtures(ctx context.Context) (*Fixtures, error) {
	now := time.Now().UnixNano()

	ws, err := db.Tx1(ctx, func(q *db.Queries) (db.Workspace, error) {
		return q.CreateWorkspace(ctx, db.CreateWorkspaceParams{
			Name:     "test-workspace",
			Slug:     fmt.Sprintf("test-workspace-%d", now),
			Settings: []byte("{}"),
			Created:  now,
			Updated:  now,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("create fixture workspace: %w", err)
	}

	return &Fixtures{Workspace: ws}, nil
}
