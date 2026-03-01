package gittokens_test

import (
	"context"
	"os"
	"testing"

	"github.com/gomantics/semantix/internal/domains/gittokens"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
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

	gt, err := gittokens.Create(ctx, gittokens.CreateParams{
		Name:     "My GitHub Token",
		Provider: gitrepo.ProviderGitHub,
		Token:    "ghp_abc1234567890",
	})

	require.NoError(t, err)
	assert.NotZero(t, gt.ID)
	assert.Equal(t, "My GitHub Token", gt.Name)
	assert.Equal(t, gitrepo.ProviderGitHub, gt.Provider)
	assert.Equal(t, "ghp_abc1234567890", gt.Token)
	assert.Equal(t, "...7890", gt.Hint)
	assert.NotZero(t, gt.Created)
}

func TestCreate_shortToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	gt, err := gittokens.Create(ctx, gittokens.CreateParams{
		Name:     "Short Token",
		Provider: gitrepo.ProviderGitHub,
		Token:    "abc",
	})

	require.NoError(t, err)
	assert.Equal(t, "", gt.Hint)
}

func TestGetByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created, err := gittokens.Create(ctx, gittokens.CreateParams{
		Name:     "GetByID Token",
		Provider: gitrepo.ProviderGitLab,
		Token:    "glpat_123456",
	})
	require.NoError(t, err)

	got, err := gittokens.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "GetByID Token", got.Name)
	assert.Equal(t, "...3456", got.Hint, "token_hint should be persisted and returned")
}

func TestGetByID_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, err := gittokens.GetByID(ctx, 999999999)
	assert.ErrorIs(t, err, gittokens.ErrNotFound)
}

func TestFindForProvider_found(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created, err := gittokens.Create(ctx, gittokens.CreateParams{
		Name:     "GitLab Token",
		Provider: gitrepo.ProviderGitLab,
		Token:    "glpat_mytoken",
	})
	require.NoError(t, err)

	got, err := gittokens.FindForProvider(ctx, gitrepo.ProviderGitLab)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, gitrepo.ProviderGitLab, got.Provider)
	assert.NotZero(t, got.ID)
	_ = created
}

func TestFindForProvider_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	got, err := gittokens.FindForProvider(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created, err := gittokens.Create(ctx, gittokens.CreateParams{
		Name:     "To Delete",
		Provider: gitrepo.ProviderGitHub,
		Token:    "ghp_todelete",
	})
	require.NoError(t, err)

	err = gittokens.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = gittokens.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, gittokens.ErrNotFound)
}

func TestDelete_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	err := gittokens.Delete(ctx, 999999999)
	assert.ErrorIs(t, err, gittokens.ErrNotFound)
}
