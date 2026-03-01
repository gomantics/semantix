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

// uniqueSlug generates a unique slug using the test name and a counter
// that is safe for parallel tests.
func uniqueSlug(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%s", prefix, testutil.UniqueID())
}

func TestCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	desc := "a test workspace"

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name:        "Test Workspace",
		Slug:        uniqueSlug(t, "create"),
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
		Slug:     uniqueSlug(t, "no-settings"),
		Settings: nil,
	})

	require.NoError(t, err)
	assert.Empty(t, ws.Settings.ExcludePatterns, "settings should default to empty")
	assert.Empty(t, ws.Settings.IncludePatterns, "settings should default to empty")
}

func TestCreate_slugConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	slug := uniqueSlug(t, "conflict")

	_, err := workspaces.Create(ctx, workspaces.CreateParams{Name: "First", Slug: slug})
	require.NoError(t, err)

	_, err = workspaces.Create(ctx, workspaces.CreateParams{Name: "Second", Slug: slug})
	assert.ErrorIs(t, err, workspaces.ErrAlreadyExists)
}

func TestGetByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "GetByID",
		Slug: uniqueSlug(t, "getbyid"),
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

func TestGetBySlug(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	slug := uniqueSlug(t, "slug")

	created, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "Slug Test",
		Slug: slug,
	})
	require.NoError(t, err)

	got, err := workspaces.GetBySlug(ctx, slug)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestGetBySlug_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, err := workspaces.GetBySlug(ctx, "nonexistent-slug-xyz-"+testutil.UniqueID())
	assert.ErrorIs(t, err, workspaces.ErrNotFound)
}

func TestList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prefix := uniqueSlug(t, "list")

	// Create 3 workspaces with unique slugs.
	for i := 0; i < 3; i++ {
		_, err := workspaces.Create(ctx, workspaces.CreateParams{
			Name: fmt.Sprintf("List WS %d", i),
			Slug: fmt.Sprintf("%s-%d", prefix, i),
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
	prefix := uniqueSlug(t, "page")

	for i := 0; i < 5; i++ {
		_, err := workspaces.Create(ctx, workspaces.CreateParams{
			Name: fmt.Sprintf("Page WS %d", i),
			Slug: fmt.Sprintf("%s-%d", prefix, i),
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
		Slug: uniqueSlug(t, "update"),
	})
	require.NoError(t, err)

	newDesc := "updated description"
	updated, err := workspaces.Update(ctx, ws.ID, workspaces.UpdateParams{
		Name:        "After Update",
		Slug:        uniqueSlug(t, "updated"),
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

func TestUpdate_slugConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ws1, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "WS1",
		Slug: uniqueSlug(t, "conflict-ws1"),
	})
	require.NoError(t, err)

	ws2, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "WS2",
		Slug: uniqueSlug(t, "conflict-ws2"),
	})
	require.NoError(t, err)

	// Attempt to update ws2's slug to ws1's slug.
	_, err = workspaces.Update(ctx, ws2.ID, workspaces.UpdateParams{
		Name:     "WS2",
		Slug:     ws1.Slug,
		Settings: nil,
	})
	assert.ErrorIs(t, err, workspaces.ErrAlreadyExists)
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "To Delete",
		Slug: uniqueSlug(t, "delete"),
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
