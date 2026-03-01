package gittokens

import (
	"context"
	"errors"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("git token not found")

func Create(ctx context.Context, params CreateParams) (*GitToken, error) {
	now := time.Now().UnixNano()

	row, err := db.Tx1(ctx, func(q *db.Queries) (db.CreateGitTokenRow, error) {
		return q.CreateGitToken(ctx, db.CreateGitTokenParams{
			Name:           params.Name,
			Provider:       params.Provider,
			TokenEncrypted: []byte(params.Token),
			Created:        now,
		})
	})
	if err != nil {
		return nil, err
	}

	return toGitToken(row.ID, row.Name, row.Provider, string(row.TokenEncrypted), now), nil
}

func GetByID(ctx context.Context, id int64) (*GitToken, error) {
	row, err := db.Query1(ctx, func(q *db.Queries) (db.GetGitTokenByIDRow, error) {
		return q.GetGitTokenByID(ctx, id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return toGitToken(row.ID, row.Name, row.Provider, string(row.TokenEncrypted), row.Created), nil
}

// FindForProvider returns the first available token for the given provider.
func FindForProvider(ctx context.Context, provider gitrepo.Provider) (*GitToken, error) {
	rows, err := db.Query1(ctx, func(q *db.Queries) ([]db.ListGitTokensByProviderRow, error) {
		return q.ListGitTokensByProvider(ctx, string(provider))
	})
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	r := rows[0]
	return toGitToken(r.ID, r.Name, r.Provider, string(r.TokenEncrypted), r.Created), nil
}

func List(ctx context.Context) ([]GitToken, error) {
	rows, err := db.Query1(ctx, func(q *db.Queries) ([]db.ListGitTokensRow, error) {
		return q.ListGitTokens(ctx)
	})
	if err != nil {
		return nil, err
	}

	tokens := make([]GitToken, len(rows))
	for i, r := range rows {
		tokens[i] = *toGitToken(r.ID, r.Name, r.Provider, string(r.TokenEncrypted), r.Created)
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

func toGitToken(id int64, name, provider, token string, created int64) *GitToken {
	hint := ""
	if len(token) >= 4 {
		hint = "..." + token[len(token)-4:]
	}

	return &GitToken{
		ID:       id,
		Name:     name,
		Provider: provider,
		Token:    token,
		Hint:     hint,
		Created:  created,
	}
}
