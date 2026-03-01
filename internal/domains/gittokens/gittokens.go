package gittokens

import (
	"context"
	"errors"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("git token not found")

func Create(ctx context.Context, params CreateParams) (*GitToken, error) {
	now := time.Now().UnixNano()

	tokenHint := pgtype.Text{}
	if len(params.Token) >= 4 {
		tokenHint = pgtype.Text{String: params.Token[len(params.Token)-4:], Valid: true}
	}

	row, err := db.Tx1(ctx, func(q *db.Queries) (db.GitToken, error) {
		return q.CreateGitToken(ctx, db.CreateGitTokenParams{
			Name:           params.Name,
			Provider:       params.Provider,
			TokenEncrypted: []byte(params.Token),
			TokenHint:      tokenHint,
			Created:        now,
		})
	})
	if err != nil {
		return nil, err
	}

	return rowToGitToken(row), nil
}

func GetByID(ctx context.Context, id int64) (*GitToken, error) {
	row, err := db.Query1(ctx, func(q *db.Queries) (db.GitToken, error) {
		return q.GetGitTokenByID(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return rowToGitToken(row), nil
}

// FindForProvider returns the first available token for the given provider.
func FindForProvider(ctx context.Context, provider gitrepo.Provider) (*GitToken, error) {
	rows, err := db.Query1(ctx, func(q *db.Queries) ([]db.GitToken, error) {
		return q.ListGitTokensByProvider(ctx, string(provider))
	})
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	return rowToGitToken(rows[0]), nil
}

func List(ctx context.Context) ([]GitToken, error) {
	rows, err := db.Query1(ctx, func(q *db.Queries) ([]db.GitToken, error) {
		return q.ListGitTokens(ctx)
	})
	if err != nil {
		return nil, err
	}

	tokens := make([]GitToken, len(rows))
	for i, r := range rows {
		tokens[i] = *rowToGitToken(r)
	}
	return tokens, nil
}

func Delete(ctx context.Context, id int64) error {
	return db.Tx(ctx, func(q *db.Queries) error {
		_, err := q.GetGitTokenByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		return q.DeleteGitToken(ctx, id)
	})
}

func rowToGitToken(r db.GitToken) *GitToken {
	hint := ""
	if r.TokenHint.Valid {
		hint = "..." + r.TokenHint.String
	}

	return &GitToken{
		ID:       r.ID,
		Name:     r.Name,
		Provider: r.Provider,
		Token:    string(r.TokenEncrypted),
		Hint:     hint,
		Created:  r.Created,
	}
}
