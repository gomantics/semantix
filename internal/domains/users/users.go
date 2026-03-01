package users

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

var (
	randomAdjectives = []string{
		"Amber", "Arctic", "Azure", "Bold", "Bright", "Calm", "Cosmic", "Crimson",
		"Crystal", "Dark", "Dawn", "Deep", "Divine", "Dusk", "Ember", "Epic",
		"Fierce", "Frosty", "Gilded", "Glacial", "Golden", "Grand", "Iron", "Jade",
		"Lunar", "Mighty", "Mystic", "Noble", "Obsidian", "Onyx", "Opal", "Phantom",
		"Polar", "Primal", "Proud", "Radiant", "Royal", "Ruby", "Sacred", "Savage",
		"Shadow", "Silent", "Silver", "Solar", "Starlit", "Steel", "Storm", "Swift",
		"Timeless", "Titan", "Twilight", "Vast", "Velvet", "Vivid", "Wild", "Wise",
	}
	randomAnimals = []string{
		"Badger", "Bear", "Bison", "Condor", "Cougar", "Crane", "Crow", "Dingo",
		"Dragon", "Eagle", "Falcon", "Fox", "Grizzly", "Hawk", "Heron", "Jaguar",
		"Kestrel", "Kodiak", "Leopard", "Lion", "Lynx", "Mammoth", "Mantis", "Marlin",
		"Mink", "Moose", "Narwhal", "Osprey", "Otter", "Owl", "Panther", "Pegasus",
		"Phoenix", "Puma", "Raven", "Rhino", "Sabre", "Salmon", "Scorpion", "Shark",
		"Snow Leopard", "Stallion", "Tiger", "Timber Wolf", "Viper", "Walrus",
		"Weasel", "Wolf", "Wolverine", "Wren",
	}
)

func randomWorkspaceName() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "Default"
	}
	hi := binary.BigEndian.Uint32(buf[:4])
	lo := binary.BigEndian.Uint32(buf[4:])
	adj := randomAdjectives[hi%uint32(len(randomAdjectives))]
	animal := randomAnimals[lo%uint32(len(randomAnimals))]
	return fmt.Sprintf("%s %s", adj, animal)
}

var (
	ErrNotFound        = errors.New("user not found")
	ErrAlreadyExists   = errors.New("user with this email already exists")
	ErrInvalidCreds    = errors.New("invalid email or password")
	ErrSessionNotFound = errors.New("session not found")
	ErrAdminExists     = errors.New("admin user already exists")
)

func Count(ctx context.Context) (int64, error) {
	return db.Query1(ctx, func(q *db.Queries) (int64, error) {
		return q.CountUsers(ctx)
	})
}

func Create(ctx context.Context, params CreateParams) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixNano()
	dbUser, err := db.Tx1(ctx, func(q *db.Queries) (db.User, error) {
		_, err := q.GetUserByEmail(ctx, params.Email)
		if err == nil {
			return db.User{}, ErrAlreadyExists
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, err
		}

		return q.CreateUser(ctx, db.CreateUserParams{
			Email:        params.Email,
			PasswordHash: string(hash),
			Created:      now,
			Updated:      now,
		})
	})
	if err != nil {
		return nil, err
	}

	return toUser(dbUser), nil
}

// CreateFirst atomically creates the first (admin) user. Returns ErrAdminExists
// if any user already exists. The count check and insert run in a single
// transaction to avoid TOCTOU races.
func CreateFirst(ctx context.Context, params CreateParams) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixNano()
	dbUser, err := db.Tx1(ctx, func(q *db.Queries) (db.User, error) {
		count, err := q.CountUsers(ctx)
		if err != nil {
			return db.User{}, err
		}
		if count > 0 {
			return db.User{}, ErrAdminExists
		}

		return q.CreateUser(ctx, db.CreateUserParams{
			Email:        params.Email,
			PasswordHash: string(hash),
			Created:      now,
			Updated:      now,
		})
	})
	if err != nil {
		return nil, err
	}

	return toUser(dbUser), nil
}

type SignupResult struct {
	User  *User
	Token string
}

// Signup atomically creates the first admin user, a session, and the default
// workspace in a single transaction. Returns ErrAdminExists if any user exists.
func Signup(ctx context.Context, params CreateParams) (*SignupResult, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixNano()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(raw)

	result, err := db.Tx1(ctx, func(q *db.Queries) (SignupResult, error) {
		count, err := q.CountUsers(ctx)
		if err != nil {
			return SignupResult{}, err
		}
		if count > 0 {
			return SignupResult{}, ErrAdminExists
		}

		dbUser, err := q.CreateUser(ctx, db.CreateUserParams{
			Email:        params.Email,
			PasswordHash: string(hash),
			Created:      now,
			Updated:      now,
		})
		if err != nil {
			return SignupResult{}, err
		}

		_, err = q.CreateSession(ctx, db.CreateSessionParams{
			UserID:    dbUser.ID,
			Token:     token,
			Created:   now,
			ExpiresAt: pgtype.Int8{Valid: false},
		})
		if err != nil {
			return SignupResult{}, err
		}

		defaultSettings, err := json.Marshal(workspaces.DefaultSettings())
		if err != nil {
			return SignupResult{}, err
		}

		_, err = q.CreateWorkspace(ctx, db.CreateWorkspaceParams{
			Name:        randomWorkspaceName(),
			Description: pgtype.Text{Valid: false},
			Settings:    defaultSettings,
			Created:     now,
			Updated:     now,
		})
		if err != nil {
			return SignupResult{}, err
		}

		return SignupResult{User: toUser(dbUser), Token: token}, nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func Login(ctx context.Context, email, password string) (*User, error) {
	dbUser, err := db.Query1(ctx, func(q *db.Queries) (db.User, error) {
		return q.GetUserByEmail(ctx, email)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCreds
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCreds
	}

	return toUser(dbUser), nil
}

func CreateSession(ctx context.Context, userID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)

	now := time.Now().UnixNano()
	_, err := db.Tx1(ctx, func(q *db.Queries) (db.Session, error) {
		return q.CreateSession(ctx, db.CreateSessionParams{
			UserID:    userID,
			Token:     token,
			Created:   now,
			ExpiresAt: pgtype.Int8{Valid: false}, // never expires
		})
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

func GetUserByToken(ctx context.Context, token string) (*User, error) {
	row, err := db.Query1(ctx, func(q *db.Queries) (db.GetSessionByTokenRow, error) {
		return q.GetSessionByToken(ctx, db.GetSessionByTokenParams{
			Token: token,
			Now:   pgtype.Int8{Int64: time.Now().UnixNano(), Valid: true},
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	return &User{
		ID:    row.UserID,
		Email: row.Email,
	}, nil
}

func DeleteSession(ctx context.Context, token string) error {
	return db.Tx(ctx, func(q *db.Queries) error {
		return q.DeleteSession(ctx, token)
	})
}

func toUser(u db.User) *User {
	return &User{
		ID:      u.ID,
		Email:   u.Email,
		Created: u.Created,
		Updated: u.Updated,
	}
}
