package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

const (
	ScopeActivation    = "activation"
	ScopeAuthorization = "access"
	ScopeRefresh       = "refresh"
)

// Store wraps Queries and provides a pool for transaction management.
// High-level methods that compose multiple queries atomically live here.
// Simple single-query operations are available via the embedded *Queries.
type Store struct {
	*Queries
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{Queries: New(pool), pool: pool}
}

func (s *Store) withTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetJarsByUser wraps the sqlc query to accept a plain int64 instead of pgtype.Int8.
func (s *Store) GetJarsByUser(ctx context.Context, userID int64) ([]Scrolljar, error) {
	return s.Queries.GetJarsByUser(ctx, pgtype.Int8{Int64: userID, Valid: true})
}

// InsertJar inserts a new jar, retrying on primary key collision.
func (s *Store) InsertJar(ctx context.Context, arg InsertJarParams) (Scrolljar, error) {
	return insertJarWithRetry(ctx, s.Queries, arg)
}

// InsertScroll inserts a new scroll, retrying on primary key collision.
func (s *Store) InsertScroll(ctx context.Context, arg InsertScrollParams) (Scroll, error) {
	return insertScrollWithRetry(ctx, s.Queries, arg)
}

// UpdateScroll maps pgx.ErrNoRows to ErrEditConflict for optimistic locking.
func (s *Store) UpdateScroll(ctx context.Context, arg UpdateScrollParams) (pgtype.Timestamptz, error) {
	ts, err := s.Queries.UpdateScroll(ctx, arg)
	if errors.Is(err, pgx.ErrNoRows) {
		return ts, ErrEditConflict
	}
	return ts, err
}

// SetScrollUploaded maps pgx.ErrNoRows to ErrEditConflict for optimistic locking.
func (s *Store) SetScrollUploaded(ctx context.Context, arg SetScrollUploadedParams) (pgtype.Timestamptz, error) {
	ts, err := s.Queries.SetScrollUploaded(ctx, arg)
	if errors.Is(err, pgx.ErrNoRows) {
		return ts, ErrEditConflict
	}
	return ts, err
}

// InsertUser maps duplicate email errors.
func (s *Store) InsertUser(ctx context.Context, arg InsertUserParams) (UserAccount, error) {
	user, err := s.Queries.InsertUser(ctx, arg)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "user_account_email_key" {
		return user, ErrDuplicateUser
	}
	return user, err
}

// CreateJarWithScrolls atomically creates a jar and its initial scrolls.
// Returns the jar, the scrolls, and the upload token for each scroll in order.
func (s *Store) CreateJarWithScrolls(ctx context.Context, jarArg InsertJarParams, scrollArgs []InsertScrollParams) (Scrolljar, []Scroll, error) {
	var jar Scrolljar
	var scrolls []Scroll

	err := s.withTx(ctx, func(q *Queries) error {
		var err error
		jar, err = insertJarWithRetry(ctx, q, jarArg)
		if err != nil {
			return err
		}
		for _, sa := range scrollArgs {
			sa.JarID = jar.ID
			scroll, err := insertScrollWithRetry(ctx, q, sa)
			if err != nil {
				return err
			}
			scrolls = append(scrolls, scroll)
		}
		return nil
	})
	return jar, scrolls, err
}

// CreateUserWithActivationToken atomically inserts a user and an activation token.
// Returns the user and the plain-text token.
func (s *Store) CreateUserWithActivationToken(ctx context.Context, arg InsertUserParams) (UserAccount, string, error) {
	var user UserAccount
	tokenText, tokenHash := newToken()
	expiry := time.Now().Add(5 * time.Minute)

	err := s.withTx(ctx, func(q *Queries) error {
		var err error
		user, err = q.InsertUser(ctx, arg)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "user_account_email_key" {
				return ErrDuplicateUser
			}
			return err
		}
		return q.UpsertToken(ctx, UpsertTokenParams{
			TokenHash: tokenHash[:],
			UserID:    user.ID,
			ExpiresAt: pgtype.Timestamptz{Time: expiry, Valid: true},
			Scope:     ScopeActivation,
		})
	})
	return user, tokenText, err
}

// ActivateUser atomically validates an activation token and marks the user as active.
func (s *Store) ActivateUser(ctx context.Context, tokenHash []byte) (UserAccount, error) {
	var user UserAccount

	err := s.withTx(ctx, func(q *Queries) error {
		token, err := q.GetTokenByHash(ctx, tokenHash)
		if err != nil {
			return err
		}
		if token.Scope != ScopeActivation {
			return pgx.ErrNoRows
		}

		user, err = q.GetUserByID(ctx, token.UserID)
		if err != nil {
			return err
		}

		if _, err = q.UpdateUser(ctx, UpdateUserParams{
			Username:     user.Username,
			Email:        user.Email,
			Activated:    true,
			PasswordHash: user.PasswordHash,
			ID:           user.ID,
			UpdatedAt:    user.UpdatedAt,
		}); err != nil {
			return err
		}
		user.Activated = true
		return q.DeleteTokenByHash(ctx, tokenHash)
	})
	return user, err
}

// AuthTokenResult holds the results of UpsertAuthTokens.
type AuthTokenResult struct {
	AuthText      string
	RefreshText   string
	AuthExpiry    time.Time
	RefreshExpiry time.Time
}

// UpsertAuthTokens atomically creates/replaces auth and refresh tokens for a user.
func (s *Store) UpsertAuthTokens(ctx context.Context, userID int64) (AuthTokenResult, error) {
	authText, authHash := newToken()
	refreshText, refreshHash := newToken()
	result := AuthTokenResult{
		AuthText:      authText,
		RefreshText:   refreshText,
		AuthExpiry:    time.Now().Add(24 * time.Hour),
		RefreshExpiry: time.Now().Add(30 * 24 * time.Hour),
	}

	err := s.withTx(ctx, func(q *Queries) error {
		if err := q.UpsertToken(ctx, UpsertTokenParams{
			TokenHash: authHash[:],
			UserID:    userID,
			ExpiresAt: pgtype.Timestamptz{Time: result.AuthExpiry, Valid: true},
			Scope:     ScopeAuthorization,
		}); err != nil {
			return err
		}
		return q.UpsertToken(ctx, UpsertTokenParams{
			TokenHash: refreshHash[:],
			UserID:    userID,
			ExpiresAt: pgtype.Timestamptz{Time: result.RefreshExpiry, Valid: true},
			Scope:     ScopeRefresh,
		})
	})
	return result, err
}

// CreateActivationToken upserts a new activation token for a user.
// Returns the plain-text token and its expiry.
func (s *Store) CreateActivationToken(ctx context.Context, userID int64) (string, time.Time, error) {
	tokenText, tokenHash := newToken()
	expiry := time.Now().Add(5 * time.Minute)
	err := s.Queries.UpsertToken(ctx, UpsertTokenParams{
		TokenHash: tokenHash[:],
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiry, Valid: true},
		Scope:     ScopeActivation,
	})
	return tokenText, expiry, err
}

// newToken generates a random token text and its SHA-256 hash.
func newToken() (string, [32]byte) {
	text := rand.Text()
	return text, sha256.Sum256([]byte(text))
}

func insertJarWithRetry(ctx context.Context, q *Queries, arg InsertJarParams) (Scrolljar, error) {
	for {
		id, err := gonanoid.Generate(base62Chars, 8)
		if err != nil {
			return Scrolljar{}, err
		}
		arg.ID = id
		jar, err := q.InsertJar(ctx, arg)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "scrolljar_pkey" {
				continue
			}
			return Scrolljar{}, err
		}
		return jar, nil
	}
}

func insertScrollWithRetry(ctx context.Context, q *Queries, arg InsertScrollParams) (Scroll, error) {
	for {
		id, err := gonanoid.Generate(base62Chars, 8)
		if err != nil {
			return Scroll{}, err
		}
		arg.ID = id
		scroll, err := q.InsertScroll(ctx, arg)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "scroll_pkey" {
				continue
			}
			return Scroll{}, err
		}
		return scroll, nil
	}
}
