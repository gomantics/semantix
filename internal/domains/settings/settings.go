package settings

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/libs/crypto"
)

const (
	KeyEncryptionKey = "encryption_key"
	KeyOpenAIAPIKey  = "openai_api_key"
)

var ErrNotFound = errors.New("setting not found")

type state struct {
	encryptionKey []byte
	openAIKey     string // decrypted plaintext, empty if not configured
}

var (
	mu      sync.Mutex
	current *state
)

func load(ctx context.Context) (*state, error) {
	rows, err := db.Query1(ctx, func(q *db.Queries) ([]db.Setting, error) {
		return q.ListSettings(ctx)
	})
	if err != nil {
		return nil, err
	}

	s := &state{}
	for _, r := range rows {
		switch r.Key {
		case KeyEncryptionKey:
			s.encryptionKey = r.Value
		case KeyOpenAIAPIKey:
			if s.encryptionKey != nil {
				plaintext, err := crypto.Decrypt(r.Value, s.encryptionKey)
				if err == nil {
					s.openAIKey = string(plaintext)
				}
			}
		}
	}
	return s, nil
}

func get() *state {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		return current
	}
	s, err := load(context.Background())
	if err != nil || s == nil {
		return &state{}
	}
	current = s
	return current
}

func reset() {
	mu.Lock()
	current = nil
	mu.Unlock()
}

// Reload forces the next access to re-read from the database. Intended for
// tests where settings are written outside the normal API.
func Reload() {
	reset()
}

// EncryptionKey returns the encryption key. Returns nil if no user has signed
// up yet.
func EncryptionKey() []byte {
	return get().encryptionKey
}

// IsSetupComplete returns true if the app has been initialised.
func IsSetupComplete() bool {
	return get().encryptionKey != nil
}

// IsOpenAIConfigured returns true if an OpenAI API key has been stored.
func IsOpenAIConfigured() bool {
	return get().openAIKey != ""
}

// GetOpenAIKey returns the stored OpenAI API key in plaintext.
func GetOpenAIKey() (string, error) {
	s := get()
	if s.encryptionKey == nil {
		return "", errors.New("encryption key not available")
	}
	if s.openAIKey == "" {
		return "", ErrNotFound
	}
	return s.openAIKey, nil
}

// SetOpenAIKey encrypts and persists the OpenAI API key.
func SetOpenAIKey(ctx context.Context, apiKey string) error {
	k := EncryptionKey()
	if k == nil {
		return errors.New("encryption key not available")
	}

	encrypted, err := crypto.Encrypt([]byte(apiKey), k)
	if err != nil {
		return err
	}

	_, err = db.Tx1(ctx, func(q *db.Queries) (db.Setting, error) {
		return q.UpsertSetting(ctx, db.UpsertSettingParams{
			Key:      KeyOpenAIAPIKey,
			Value:    encrypted,
			IsSecret: true,
			Updated:  time.Now().UnixNano(),
		})
	})
	if err != nil {
		return err
	}

	reset()
	return nil
}

// GenerateEncryptionKey generates a 32-byte random encryption key and stores
// it in the settings table using the provided *db.Queries (so it participates
// in the signup transaction).
func GenerateEncryptionKey(ctx context.Context, q *db.Queries) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}

	if err := SetWithQueries(ctx, q, KeyEncryptionKey, key, false); err != nil {
		return err
	}

	reset()
	return nil
}

// SetWithQueries writes a setting inside an existing transaction.
func SetWithQueries(ctx context.Context, q *db.Queries, key string, value []byte, isSecret bool) error {
	_, err := q.UpsertSetting(ctx, db.UpsertSettingParams{
		Key:      key,
		Value:    value,
		IsSecret: isSecret,
		Updated:  time.Now().UnixNano(),
	})
	return err
}

// Setting is the API representation of a persisted setting.
type Setting struct {
	Key      string
	Value    string
	IsSecret bool
	Updated  int64
}

// List returns all settings visible via the API. Secret values are masked and
// the encryption key is excluded.
func List(ctx context.Context) ([]Setting, error) {
	rows, err := db.Query1(ctx, func(q *db.Queries) ([]db.Setting, error) {
		return q.ListSettings(ctx)
	})
	if err != nil {
		return nil, err
	}

	out := make([]Setting, 0, len(rows))
	for _, r := range rows {
		if r.Key == KeyEncryptionKey {
			continue
		}
		value := string(r.Value)
		if r.IsSecret {
			value = "***"
		}
		out = append(out, Setting{
			Key:      r.Key,
			Value:    value,
			IsSecret: r.IsSecret,
			Updated:  r.Updated,
		})
	}
	return out, nil
}
