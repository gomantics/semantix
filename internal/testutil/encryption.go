package testutil

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/domains/settings"
)

// WithEncryptionKey returns an Option that generates a random encryption key
// and stores it in the settings table. Use this in TestMain for tests that
// interact with git tokens or other encrypted data but do not call WithAdminUser.
func WithEncryptionKey() Option {
	return func() Teardown {
		return withEncryptionKey()
	}
}

func withEncryptionKey() Teardown {
	ctx := context.Background()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(fmt.Sprintf("testutil: generate encryption key: %v", err))
	}

	err := db.Tx(ctx, func(q *db.Queries) error {
		return settings.SetWithQueries(ctx, q, settings.KeyEncryptionKey, key, false)
	})
	if err != nil {
		panic(fmt.Sprintf("testutil: store encryption key: %v", err))
	}

	// Reset the lazy-load cache so the next EncryptionKey() call picks up the
	// key we just wrote.
	settings.Reload()

	return func() {}
}
