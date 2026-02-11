package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
)

// SessionRepository defines the data access contract for sessions.
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByToken(ctx context.Context, tokenHash string) (*model.Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type sessionRepo struct {
	db *sql.DB
}

// NewSessionRepo creates a new SessionRepository backed by the given database.
func NewSessionRepo(db *sql.DB) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.RefreshTokenHash,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repo: create session: %w", err)
	}
	return nil
}

func (r *sessionRepo) GetByToken(ctx context.Context, tokenHash string) (*model.Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, expires_at, created_at, revoked_at
		FROM sessions
		WHERE refresh_token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > $2
	`
	s := &model.Session{}
	err := r.db.QueryRowContext(ctx, query, tokenHash, time.Now().UTC()).Scan(
		&s.ID,
		&s.UserID,
		&s.RefreshTokenHash,
		&s.ExpiresAt,
		&s.CreatedAt,
		&s.RevokedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: get session by token: %w", err)
	}
	return s, nil
}

func (r *sessionRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE sessions SET revoked_at = $1
		WHERE id = $2 AND revoked_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("repo: revoke session: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *sessionRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE sessions SET revoked_at = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("repo: revoke all sessions for user: %w", err)
	}
	return nil
}
