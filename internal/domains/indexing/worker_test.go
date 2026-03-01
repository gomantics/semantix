package indexing_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/domains/indexing"
	"github.com/gomantics/semantix/internal/domains/repos"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
	"github.com/gomantics/semantix/internal/libs/openai"
	"github.com/gomantics/semantix/internal/qdrant"
	"github.com/gomantics/semantix/internal/testutil"
	pb "github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	// Use a temp directory for cloned repos so tests don't pollute the workspace.
	cloneDir, err := os.MkdirTemp("", "semantix-test-clone-*")
	if err != nil {
		panic(fmt.Sprintf("create clone dir: %v", err))
	}
	os.Setenv("CONFIG_INDEXING_CLONE_DIR", cloneDir)

	main := testutil.Main(m,
		testutil.WithPostgres(),
		testutil.WithQdrant(),
	)
	code := main.Run()
	os.RemoveAll(cloneDir)
	os.Exit(code)
}

// makeWorkspaceAndRepo creates a workspace and a pending repo for testing.
func makeWorkspaceAndRepo(t *testing.T) (int64, repos.Repo) {
	t.Helper()
	ctx := context.Background()

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name: "Worker Test Workspace",
		Slug: fmt.Sprintf("worker-test-%s-%d", testutil.UniqueID(), time.Now().UnixNano()),
	})
	require.NoError(t, err)

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: ws.ID,
		URL:         "https://github.com/test/repo",
		Branch:      "main",
	})
	require.NoError(t, err)

	return ws.ID, *repo
}

// goFiles contains two small Go source files for the fake cloner.
var goFiles = map[string][]byte{
	"main.go": []byte(`package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`),
	"utils.go": []byte(`package main

func add(a, b int) int {
	return a + b
}
`),
}

func makeWorker(t *testing.T, cloner gitrepo.Cloner) *indexing.Worker {
	t.Helper()
	l := zap.NewNop()
	return indexing.NewWorker(l,
		indexing.WithCloner(cloner),
		indexing.WithEmbedder(&openai.FakeEmbedder{}),
	)
}

func countRepoPoints(t *testing.T, repoID int64) uint64 {
	t.Helper()
	ctx := context.Background()

	count, err := qdrant.CountPoints(ctx, &pb.Filter{
		Must: []*pb.Condition{
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "repo_id",
						Match: &pb.Match{
							MatchValue: &pb.Match_Integer{Integer: repoID},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	return count
}

func getIndexRunForRepo(t *testing.T, repoID int64) db.IndexRun {
	t.Helper()
	ctx := context.Background()

	runs, err := db.Query1(ctx, func(q *db.Queries) ([]db.IndexRun, error) {
		return q.ListIndexRunsByRepo(ctx, db.ListIndexRunsByRepoParams{
			RepoID: repoID,
			Limit:  1,
			Offset: 0,
		})
	})
	require.NoError(t, err)
	require.NotEmpty(t, runs, "expected at least one index run")
	return runs[0]
}

func TestWorker_Process_happyPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, repo := makeWorkspaceAndRepo(t)

	worker := makeWorker(t, &gitrepo.FakeCloner{Files: goFiles})
	worker.Process(ctx, repo)

	// Repo status should be ready.
	updated, err := repos.GetByID(ctx, repo.ID)
	require.NoError(t, err)
	assert.Equal(t, repos.StatusReady, updated.Status)
	assert.NotNil(t, updated.IndexedAt)

	// Index run should be completed with stats.
	run := getIndexRunForRepo(t, repo.ID)
	assert.Equal(t, "completed", run.Status)
	assert.Greater(t, run.FilesProcessed, int32(0))
	assert.Greater(t, run.ChunksCreated, int32(0))
	assert.Greater(t, run.EmbeddingsGenerated, int32(0))
	assert.NotNil(t, run.CompletedAt)

	// Qdrant should have points for this repo.
	pointCount := countRepoPoints(t, repo.ID)
	assert.Greater(t, pointCount, uint64(0))
}

func TestWorker_Process_fileUnchangedCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, repo := makeWorkspaceAndRepo(t)

	worker := makeWorker(t, &gitrepo.FakeCloner{Files: goFiles})

	// First run - indexes everything.
	worker.Process(ctx, repo)

	run1 := getIndexRunForRepo(t, repo.ID)
	assert.Equal(t, "completed", run1.Status)
	firstEmbeddings := run1.EmbeddingsGenerated

	// Reset repo to pending for second run.
	_, err := repos.UpdateStatus(ctx, repo.ID, repos.StatusPending, nil)
	require.NoError(t, err)
	updatedRepo, err := repos.GetByID(ctx, repo.ID)
	require.NoError(t, err)

	// Second run with identical files.
	worker.Process(ctx, *updatedRepo)

	// Get the most recent run (the second one).
	run2 := getIndexRunForRepo(t, repo.ID)
	assert.Equal(t, "completed", run2.Status)

	// On the second run, all files are cached (same content hash) so no new
	// embeddings should be generated.
	assert.Equal(t, int32(0), run2.EmbeddingsGenerated,
		"second run with identical files should skip embedding generation (files cached)")
	_ = firstEmbeddings
}

func TestWorker_Process_cloneFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, repo := makeWorkspaceAndRepo(t)

	cloneErr := errors.New("authentication required")
	worker := makeWorker(t, &gitrepo.FakeCloner{Err: cloneErr})
	worker.Process(ctx, repo)

	// Repo status should be error.
	updated, err := repos.GetByID(ctx, repo.ID)
	require.NoError(t, err)
	assert.Equal(t, repos.StatusError, updated.Status)
	assert.NotNil(t, updated.ErrorMessage)
	assert.Contains(t, *updated.ErrorMessage, "authentication required")

	// Index run should be failed.
	run := getIndexRunForRepo(t, repo.ID)
	assert.Equal(t, "failed", run.Status)
	assert.NotNil(t, run.ErrorMessage)

	// No Qdrant points should exist for this repo.
	pointCount := countRepoPoints(t, repo.ID)
	assert.Equal(t, uint64(0), pointCount)
}

func TestWorker_Process_statusTransitions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, repo := makeWorkspaceAndRepo(t)

	// Verify repo starts pending.
	initial, err := repos.GetByID(ctx, repo.ID)
	require.NoError(t, err)
	assert.Equal(t, repos.StatusPending, initial.Status)

	worker := makeWorker(t, &gitrepo.FakeCloner{Files: goFiles})
	worker.Process(ctx, repo)

	// After processing, repo should be ready (not pending, cloning, or indexing).
	final, err := repos.GetByID(ctx, repo.ID)
	require.NoError(t, err)
	assert.Equal(t, repos.StatusReady, final.Status)
}
