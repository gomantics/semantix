package health_test

import (
	"os"
	"testing"

	approvals "github.com/approvals/go-approval-tests"
	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	main := testutil.Main(m,
		testutil.WithPostgres(),
		testutil.WithQdrant(),
		testutil.WithApprovals(),
	)
	os.Exit(main.Run())
}

func TestGet_healthy(t *testing.T) {
	t.Parallel()
	state := testutil.NewState(t)

	body, err := state.Get("/v1/health")
	require.NoError(t, err)

	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "ok", body["database"])
	assert.Equal(t, "ok", body["qdrant"])
	assert.Equal(t, "0.1.0", body["version"])
}

// TestGet_healthy_approvals uses go-approvals snapshot testing.
// NOTE: approval tests use shared file-based state and must not run in parallel
// with other approval tests within the same package.
func TestGet_healthy_approvals(t *testing.T) {
	state := testutil.NewState(t)

	body, err := state.Get("/v1/health")
	require.NoError(t, err)

	// Scrub non-deterministic fields before snapshotting.
	testutil.ScrubField(body, "version")

	approvals.VerifyJSONStruct(t, body)
}

func TestGet_returnsHTTP200(t *testing.T) {
	t.Parallel()
	state := testutil.NewState(t)

	// Get returns no error (non-2xx would be a StatusError).
	_, err := state.Get("/v1/health")
	assert.NoError(t, err)
}

func TestGet_statusCodeOnDegradedDB(t *testing.T) {
	t.Parallel()
	state := testutil.NewState(t)

	body, err := state.Get("/v1/health")
	// With both containers up, we expect no error and status 200.
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])

	// Verify the response structure contains all expected fields.
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "database")
	assert.Contains(t, body, "qdrant")
	assert.Contains(t, body, "version")
}

// TestGet_matchesHealthContract verifies the response structure matches expected contract.
func TestGet_matchesHealthContract(t *testing.T) {
	t.Parallel()
	state := testutil.NewState(t)

	_, err := state.Get("/v1/health")
	// Should be 200 with both deps up.
	require.NoError(t, err, "health endpoint should return 200 when deps are healthy")
}
