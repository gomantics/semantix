package repos_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/domains/repos"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	main := testutil.Main(m, testutil.WithPostgres())
	os.Exit(main.Run())
}

// makeWorkspace creates an isolated workspace for a single test.
func makeWorkspace(t *testing.T) int64 {
	t.Helper()
	ctx := context.Background()
	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "Repo Test Workspace",
	})
	require.NoError(t, err)
	return ws.ID
}

func TestCreate_defaultsBranchToMain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wsID := makeWorkspace(t)

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/repo",
	})

	require.NoError(t, err)
	assert.Equal(t, "main", repo.Branch)
	assert.Equal(t, repos.StatusPending, repo.Status)
	assert.NotZero(t, repo.ID)
}

func TestCreate_customBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wsID := makeWorkspace(t)

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/repo",
		Branch:      "develop",
	})

	require.NoError(t, err)
	assert.Equal(t, "develop", repo.Branch)
}

func TestGetByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wsID := makeWorkspace(t)

	created, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/getbyid",
	})
	require.NoError(t, err)

	got, err := repos.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.URL, got.URL)
}

func TestGetByID_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, err := repos.GetByID(ctx, 999999999)
	assert.ErrorIs(t, err, repos.ErrNotFound)
}

func TestUpdateStatus_transitions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wsID := makeWorkspace(t)

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/transitions",
	})
	require.NoError(t, err)
	assert.Equal(t, repos.StatusPending, repo.Status)

	transitions := []repos.Status{
		repos.StatusCloning,
		repos.StatusIndexing,
		repos.StatusReady,
	}

	for _, status := range transitions {
		updated, err := repos.UpdateStatus(ctx, repo.ID, status, nil)
		require.NoError(t, err, "transition to %s failed", status)
		assert.Equal(t, status, updated.Status)
	}

	// Ready status should set indexed_at.
	final, err := repos.GetByID(ctx, repo.ID)
	require.NoError(t, err)
	assert.NotNil(t, final.IndexedAt)
}

func TestUpdateStatus_error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wsID := makeWorkspace(t)

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/error-status",
	})
	require.NoError(t, err)

	errMsg := "clone failed: connection refused"
	updated, err := repos.UpdateStatus(ctx, repo.ID, repos.StatusError, &errMsg)
	require.NoError(t, err)
	assert.Equal(t, repos.StatusError, updated.Status)
	assert.Equal(t, &errMsg, updated.ErrorMessage)
}

func TestList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Each test gets its own workspace so List results are fully isolated.
	wsID := makeWorkspace(t)

	for i := 0; i < 3; i++ {
		_, err := repos.Create(ctx, repos.CreateParams{
			WorkspaceID: wsID,
			URL:         fmt.Sprintf("https://github.com/org/repo-%d", i),
		})
		require.NoError(t, err)
	}

	result, err := repos.List(ctx, repos.ListParams{
		WorkspaceID: wsID,
		Limit:       100,
	})
	require.NoError(t, err)
	// Exact count is safe because List is scoped to this workspace.
	assert.Equal(t, int64(3), result.Total)
	assert.Len(t, result.Repos, 3)
}

func TestListPending(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Own workspace so we can create known pending repos in isolation.
	wsID := makeWorkspace(t)

	// Create 2 pending repos.
	pendingIDs := make(map[int64]bool)
	for i := 0; i < 2; i++ {
		r, err := repos.Create(ctx, repos.CreateParams{
			WorkspaceID: wsID,
			URL:         fmt.Sprintf("https://github.com/org/pending-%s-%d", testutil.UniqueID(), i),
		})
		require.NoError(t, err)
		pendingIDs[r.ID] = true
	}

	// Create 1 repo and advance it past pending.
	notPending, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/not-pending-" + testutil.UniqueID(),
	})
	require.NoError(t, err)
	_, err = repos.UpdateStatus(ctx, notPending.ID, repos.StatusReady, nil)
	require.NoError(t, err)

	// ListPending is global, so filter down to our workspace's repos.
	pending, err := repos.ListPending(ctx, 1000)
	require.NoError(t, err)

	// Verify our 2 pending repos appear in the results.
	foundPending := 0
	for _, r := range pending {
		assert.Equal(t, repos.StatusPending, r.Status)
		if pendingIDs[r.ID] {
			foundPending++
		}
		// Our advanced repo must not appear as pending.
		assert.NotEqual(t, notPending.ID, r.ID)
	}
	assert.Equal(t, 2, foundPending, "expected both our pending repos to appear in ListPending")
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wsID := makeWorkspace(t)

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/to-delete",
	})
	require.NoError(t, err)

	err = repos.Delete(ctx, repo.ID)
	require.NoError(t, err)

	_, err = repos.GetByID(ctx, repo.ID)
	assert.ErrorIs(t, err, repos.ErrNotFound)
}

func TestDelete_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	err := repos.Delete(ctx, 999999999)
	assert.ErrorIs(t, err, repos.ErrNotFound)
}

// TestIndexRunCreation verifies index_run records can be created for repos.
func TestIndexRunCreation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wsID := makeWorkspace(t)

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wsID,
		URL:         "https://github.com/org/index-run-test",
	})
	require.NoError(t, err)

	now := time.Now().UnixNano()
	run, err := db.Tx1(ctx, func(q *db.Queries) (db.IndexRun, error) {
		return q.CreateIndexRun(ctx, db.CreateIndexRunParams{
			RepoID:    repo.ID,
			Status:    "running",
			StartedAt: now,
		})
	})
	require.NoError(t, err)
	assert.NotZero(t, run.ID)
	assert.Equal(t, "running", run.Status)
}
