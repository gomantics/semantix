package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// AdminCreds holds the email and password of the single admin user created
// by WithAdminUser. Tests that need an authenticated session should use these
// credentials via NewAuthState, which logs in using them.
var AdminCreds struct {
	Email    string
	Password string
}

// WithAdminUser returns an Option that creates the single admin user directly
// in the database (bypassing the signup endpoint's one-user guard). It must
// be composed after WithPostgres() so the DB pool is ready.
func WithAdminUser() Option {
	return func() Teardown {
		return withAdminUser()
	}
}

func withAdminUser() Teardown {
	ctx := context.Background()

	email := fmt.Sprintf("admin-%s@test.com", UniqueID())
	password := "testpassword123"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		panic(fmt.Sprintf("testutil: hash admin password: %v", err))
	}

	now := time.Now().UnixNano()
	_, err = db.Tx1(ctx, func(q *db.Queries) (db.User, error) {
		return q.CreateUser(ctx, db.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			Created:      now,
			Updated:      now,
		})
	})
	if err != nil {
		panic(fmt.Sprintf("testutil: create admin user: %v", err))
	}

	AdminCreds.Email = email
	AdminCreds.Password = password

	return func() {}
}
