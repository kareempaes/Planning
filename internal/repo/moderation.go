package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
)

// ModerationRepository defines the data access contract for moderation.
type ModerationRepository interface {
	Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
	Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
	ListBlocked(ctx context.Context, blockerID uuid.UUID) ([]model.BlockedUser, error)
	IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error)
	CreateReport(ctx context.Context, report *model.Report) error
}

type moderationRepo struct {
	db *sql.DB
}

// NewModerationRepo creates a new ModerationRepository backed by the given database.
func NewModerationRepo(db *sql.DB) ModerationRepository {
	return &moderationRepo{db: db}
}

func (r *moderationRepo) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	query := `
		INSERT INTO blocked_users (id, blocker_id, blocked_id)
		VALUES (gen_random_uuid(), $1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("repo: block user: %w", err)
	}
	return nil
}

func (r *moderationRepo) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	query := `
		DELETE FROM blocked_users
		WHERE blocker_id = $1 AND blocked_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("repo: unblock user: %w", err)
	}
	return nil
}

func (r *moderationRepo) ListBlocked(ctx context.Context, blockerID uuid.UUID) ([]model.BlockedUser, error) {
	query := `
		SELECT id, blocker_id, blocked_id, created_at
		FROM blocked_users
		WHERE blocker_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, blockerID)
	if err != nil {
		return nil, fmt.Errorf("repo: list blocked: %w", err)
	}
	defer rows.Close()

	var blocked []model.BlockedUser
	for rows.Next() {
		var b model.BlockedUser
		if err := rows.Scan(&b.ID, &b.BlockerID, &b.BlockedID, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("repo: scan blocked user: %w", err)
		}
		blocked = append(blocked, b)
	}
	return blocked, rows.Err()
}

func (r *moderationRepo) IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM blocked_users
			WHERE blocker_id = $1 AND blocked_id = $2
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, blockerID, blockedID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repo: is blocked: %w", err)
	}
	return exists, nil
}

func (r *moderationRepo) CreateReport(ctx context.Context, report *model.Report) error {
	query := `
		INSERT INTO reports (id, reporter_id, target_type, target_id, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		report.ID,
		report.ReporterID,
		report.TargetType,
		report.TargetID,
		report.Reason,
		report.Status,
		report.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return model.ErrConflict
		}
		return fmt.Errorf("repo: create report: %w", err)
	}
	return nil
}
