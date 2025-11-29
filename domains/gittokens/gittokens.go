package gittokens

import (
	"context"
	"errors"
	"time"

	"github.com/gomantics/semantix/db"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound      = errors.New("git token not found")
	ErrAlreadyExists = errors.New("git token with this name already exists for provider")
)

// Create creates a new git token
func Create(ctx context.Context, params CreateParams) (*GitToken, error) {
	encrypted, err := encrypt(params.Token)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	dbToken, err := db.Query1(ctx, func(q *db.Queries) (db.GitToken, error) {
		return q.CreateGitToken(ctx, db.CreateGitTokenParams{
			WorkspaceID:    params.WorkspaceID,
			Provider:       string(params.Provider),
			Name:           params.Name,
			TokenEncrypted: encrypted,
			Created:        now,
			Updated:        now,
		})
	})
	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	return toGitToken(dbToken), nil
}

// GetByID retrieves a git token by ID
func GetByID(ctx context.Context, id int64) (*GitToken, error) {
	dbToken, err := db.Query1(ctx, func(q *db.Queries) (db.GitToken, error) {
		return q.GetGitTokenByID(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toGitToken(dbToken), nil
}

// GetDecryptedToken retrieves and decrypts a git token by ID
func GetDecryptedToken(ctx context.Context, id int64) (string, error) {
	dbToken, err := db.Query1(ctx, func(q *db.Queries) (db.GitToken, error) {
		return q.GetGitTokenByID(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return decrypt(dbToken.TokenEncrypted)
}

// ListByWorkspace retrieves all git tokens for a workspace
func ListByWorkspace(ctx context.Context, params ListParams) ([]GitToken, error) {
	var dbTokens []db.GitToken
	var err error

	err = db.Query(ctx, func(q *db.Queries) error {
		if params.Provider != nil {
			dbTokens, err = q.ListGitTokensByWorkspaceAndProvider(ctx, db.ListGitTokensByWorkspaceAndProviderParams{
				WorkspaceID: params.WorkspaceID,
				Provider:    string(*params.Provider),
			})
		} else {
			dbTokens, err = q.ListGitTokensByWorkspace(ctx, params.WorkspaceID)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	tokens := make([]GitToken, len(dbTokens))
	for i, dbToken := range dbTokens {
		tokens[i] = *toGitToken(dbToken)
	}

	return tokens, nil
}

// Update updates a git token
func Update(ctx context.Context, id int64, params UpdateParams) (*GitToken, error) {
	// Get existing token to preserve encrypted value if not updating
	existing, err := db.Query1(ctx, func(q *db.Queries) (db.GitToken, error) {
		return q.GetGitTokenByID(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	encrypted := existing.TokenEncrypted
	if params.Token != "" {
		encrypted, err = encrypt(params.Token)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().Unix()
	dbToken, err := db.Query1(ctx, func(q *db.Queries) (db.GitToken, error) {
		return q.UpdateGitToken(ctx, db.UpdateGitTokenParams{
			ID:             id,
			Name:           params.Name,
			TokenEncrypted: encrypted,
			Updated:        now,
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return toGitToken(dbToken), nil
}

// Delete removes a git token by ID
func Delete(ctx context.Context, id int64) error {
	_, err := GetByID(ctx, id)
	if err != nil {
		return err
	}

	return db.Query(ctx, func(q *db.Queries) error {
		return q.DeleteGitToken(ctx, id)
	})
}

// DeleteByWorkspace removes all git tokens for a workspace
func DeleteByWorkspace(ctx context.Context, workspaceID int64) error {
	return db.Query(ctx, func(q *db.Queries) error {
		return q.DeleteGitTokensByWorkspace(ctx, workspaceID)
	})
}

func toGitToken(dbToken db.GitToken) *GitToken {
	return &GitToken{
		ID:          dbToken.ID,
		WorkspaceID: dbToken.WorkspaceID,
		Provider:    Provider(dbToken.Provider),
		Name:        dbToken.Name,
		Created:     dbToken.Created,
		Updated:     dbToken.Updated,
	}
}

func isUniqueViolation(err error) bool {
	// Check for postgres unique violation error code 23505
	return err != nil && (errors.Is(err, pgx.ErrNoRows) == false) &&
		(err.Error() == "ERROR: duplicate key value violates unique constraint" ||
			contains(err.Error(), "duplicate key") ||
			contains(err.Error(), "23505"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
