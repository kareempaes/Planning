package repo

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
)

// UserRepository defines the data access contract for users.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, id uuid.UUID, params model.UpdateProfileParams) (*model.User, error)
	Search(ctx context.Context, query string, cursor string, limit int) (*model.Page[model.UserSearchResult], error)
}

type userRepo struct {
	db *sql.DB
}

// NewUserRepo creates a new UserRepository backed by the given database.
func NewUserRepo(db *sql.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, display_name, avatar_url, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.AvatarURL,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return model.ErrConflict
		}
		return fmt.Errorf("repo: create user: %w", err)
	}
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, status, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: get user by id: %w", err)
	}
	return user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, status, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: get user by email: %w", err)
	}
	return user, nil
}

func (r *userRepo) Update(ctx context.Context, id uuid.UUID, params model.UpdateProfileParams) (*model.User, error) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if params.DisplayName != nil {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *params.DisplayName)
		argIdx++
	}
	if params.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *params.AvatarURL)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	// Always update updated_at.
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $%d
		RETURNING id, email, password_hash, display_name, avatar_url, status, created_at, updated_at
	`, strings.Join(setClauses, ", "), argIdx)

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: update user: %w", err)
	}
	return user, nil
}

func (r *userRepo) Search(ctx context.Context, query string, cursor string, limit int) (*model.Page[model.UserSearchResult], error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	fetchLimit := limit + 1

	args := []any{}
	argIdx := 1

	whereClauses := []string{fmt.Sprintf("display_name ILIKE $%d", argIdx)}
	args = append(args, query+"%")
	argIdx++

	if cursor != "" {
		cursorName, cursorID, err := decodeCursor(cursor)
		if err != nil {
			return nil, &model.ValidationError{Field: "cursor", Message: "invalid cursor"}
		}
		whereClauses = append(whereClauses,
			fmt.Sprintf("(display_name, id) > ($%d, $%d)", argIdx, argIdx+1),
		)
		args = append(args, cursorName, cursorID)
		argIdx += 2
	}

	args = append(args, fetchLimit)

	sqlQuery := fmt.Sprintf(`
		SELECT id, display_name, avatar_url
		FROM users
		WHERE %s
		ORDER BY display_name ASC, id ASC
		LIMIT $%d
	`, strings.Join(whereClauses, " AND "), argIdx)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repo: search users: %w", err)
	}
	defer rows.Close()

	results := make([]model.UserSearchResult, 0, limit)
	for rows.Next() {
		var u model.UserSearchResult
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("repo: scan search result: %w", err)
		}
		results = append(results, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo: search rows error: %w", err)
	}

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	var nextCursor *string
	if hasMore && len(results) > 0 {
		last := results[len(results)-1]
		c := encodeCursor(last.DisplayName, last.ID)
		nextCursor = &c
	}

	return &model.Page[model.UserSearchResult]{
		Items:      results,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func decodeCursor(cursor string) (string, uuid.UUID, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return "", uuid.Nil, fmt.Errorf("invalid cursor format")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("invalid cursor uuid: %w", err)
	}
	return parts[0], id, nil
}

func encodeCursor(displayName string, id uuid.UUID) string {
	raw := displayName + "|" + id.String()
	return base64.URLEncoding.EncodeToString([]byte(raw))
}
