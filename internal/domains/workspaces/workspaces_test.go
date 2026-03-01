package workspaces_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/gomantics/semantix/internal/domains/workspaces"
	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	main := testutil.Main(m, testutil.WithPostgres())
	os.Exit(main.Run())
}

func TestCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	desc := "a test workspace"

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name:        "Test Workspace",
		Description: &desc,
		Settings: &workspaces.WorkspaceSettings{
			ExcludePatterns: []string{"docs/", "*.test.ts"},
		},
	})

	require.NoError(t, err)
	assert.NotZero(t, ws.ID)
	assert.Equal(t, "Test Workspace", ws.Name)
	assert.Equal(t, &desc, ws.Description)
	assert.Equal(t, []string{"docs/", "*.test.ts"}, ws.Settings.ExcludePatterns)
	assert.NotZero(t, ws.Created)
	assert.NotZero(t, ws.Updated)
}

func TestCreate_defaultSettings(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name:     "No Settings",
		Settings: nil,
	})

	require.NoError(t, err)
	assert.Equal(t, workspaces.DefaultSettings().ExcludePatterns, ws.Settings.ExcludePatterns, "settings should default to DefaultSettings")
	assert.Empty(t, ws.Settings.IncludePatterns, "include patterns should default to empty")
}

func TestGetByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "GetByID",
	})
	require.NoError(t, err)

	got, err := workspaces.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.Name, got.Name)
}

func TestGetByID_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, err := workspaces.GetByID(ctx, 999999999)
	assert.ErrorIs(t, err, workspaces.ErrNotFound)
}

func TestList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := workspaces.Create(ctx, workspaces.CreateParams{
			Name: fmt.Sprintf("List WS %d - %s", i, testutil.UniqueID()),
		})
		require.NoError(t, err)
	}

	// List uses a shared DB, so we can only assert >= 3, not exactly 3.
	result, err := workspaces.List(ctx, workspaces.ListParams{Limit: 100, Offset: 0})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Total, int64(3))
	assert.GreaterOrEqual(t, len(result.Workspaces), 3)
}

func TestList_pagination(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := workspaces.Create(ctx, workspaces.CreateParams{
			Name: fmt.Sprintf("Page WS %d - %s", i, testutil.UniqueID()),
		})
		require.NoError(t, err)
	}

	page1, err := workspaces.List(ctx, workspaces.ListParams{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, page1.Workspaces, 2)

	page2, err := workspaces.List(ctx, workspaces.ListParams{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, page2.Workspaces, 2)

	assert.NotEqual(t, page1.Workspaces[0].ID, page2.Workspaces[0].ID)
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "Before Update",
	})
	require.NoError(t, err)

	newDesc := "updated description"
	updated, err := workspaces.Update(ctx, ws.ID, workspaces.UpdateParams{
		Name:        "After Update",
		Description: &newDesc,
		Settings: &workspaces.WorkspaceSettings{
			IncludePatterns: []string{"*.go"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, ws.ID, updated.ID)
	assert.Equal(t, "After Update", updated.Name)
	assert.Equal(t, &newDesc, updated.Description)
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "To Delete",
	})
	require.NoError(t, err)

	err = workspaces.Delete(ctx, ws.ID)
	require.NoError(t, err)

	_, err = workspaces.GetByID(ctx, ws.ID)
	assert.ErrorIs(t, err, workspaces.ErrNotFound)
}

func TestDelete_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	err := workspaces.Delete(ctx, 999999999)
	assert.ErrorIs(t, err, workspaces.ErrNotFound)
}
