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

// ConversationRepository defines the data access contract for conversations.
type ConversationRepository interface {
	Create(ctx context.Context, convo *model.Conversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Conversation, error)
	ListByUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*model.Page[model.ConversationSummary], error)
	Update(ctx context.Context, id uuid.UUID, name string) (*model.Conversation, error)
	FindDirectBetween(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (*model.Conversation, error)
	AddParticipant(ctx context.Context, participant *model.ConversationParticipant) error
	RemoveParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error
	GetParticipants(ctx context.Context, conversationID uuid.UUID) ([]model.ConversationParticipant, error)
	IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error)
}

type conversationRepo struct {
	db *sql.DB
}

// NewConversationRepo creates a new ConversationRepository backed by the given database.
func NewConversationRepo(db *sql.DB) ConversationRepository {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) Create(ctx context.Context, convo *model.Conversation) error {
	query := `
		INSERT INTO conversations (id, type, name, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		convo.ID,
		convo.Type,
		convo.Name,
		convo.CreatedBy,
		convo.CreatedAt,
		convo.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repo: create conversation: %w", err)
	}
	return nil
}

func (r *conversationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Conversation, error) {
	query := `
		SELECT id, type, name, created_by, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`
	c := &model.Conversation{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.Type,
		&c.Name,
		&c.CreatedBy,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: get conversation by id: %w", err)
	}
	return c, nil
}

func (r *conversationRepo) ListByUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*model.Page[model.ConversationSummary], error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	fetchLimit := limit + 1

	args := []any{userID}
	argIdx := 2

	whereCursor := ""
	if cursor != "" {
		cursorTime, cursorID, err := decodeTimeCursor(cursor)
		if err != nil {
			return nil, &model.ValidationError{Field: "cursor", Message: "invalid cursor"}
		}
		whereCursor = fmt.Sprintf(" AND (c.updated_at, c.id) < ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, cursorTime, cursorID)
		argIdx += 2
	}

	args = append(args, fetchLimit)

	query := fmt.Sprintf(`
		SELECT c.id, c.type, c.name, c.updated_at
		FROM conversations c
		JOIN conversation_participants cp ON cp.conversation_id = c.id
		WHERE cp.user_id = $1 AND cp.left_at IS NULL%s
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT $%d
	`, whereCursor, argIdx)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repo: list conversations by user: %w", err)
	}
	defer rows.Close()

	results := make([]model.ConversationSummary, 0, limit)
	for rows.Next() {
		var s model.ConversationSummary
		var updatedAt time.Time
		if err := rows.Scan(&s.ID, &s.Type, &s.Name, &updatedAt); err != nil {
			return nil, fmt.Errorf("repo: scan conversation summary: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo: list conversations rows error: %w", err)
	}

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	var nextCursor *string
	if hasMore && len(results) > 0 {
		last := results[len(results)-1]
		// Re-fetch updated_at for cursor encoding.
		var lastUpdatedAt time.Time
		r.db.QueryRowContext(ctx, "SELECT updated_at FROM conversations WHERE id = $1", last.ID).Scan(&lastUpdatedAt)
		c := encodeTimeCursor(lastUpdatedAt, last.ID)
		nextCursor = &c
	}

	return &model.Page[model.ConversationSummary]{
		Items:      results,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *conversationRepo) Update(ctx context.Context, id uuid.UUID, name string) (*model.Conversation, error) {
	query := `
		UPDATE conversations
		SET name = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, type, name, created_by, created_at, updated_at
	`
	c := &model.Conversation{}
	err := r.db.QueryRowContext(ctx, query, name, time.Now().UTC(), id).Scan(
		&c.ID,
		&c.Type,
		&c.Name,
		&c.CreatedBy,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: update conversation: %w", err)
	}
	return c, nil
}

func (r *conversationRepo) FindDirectBetween(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (*model.Conversation, error) {
	query := `
		SELECT c.id, c.type, c.name, c.created_by, c.created_at, c.updated_at
		FROM conversations c
		WHERE c.type = 'direct'
		  AND EXISTS (
		    SELECT 1 FROM conversation_participants
		    WHERE conversation_id = c.id AND user_id = $1 AND left_at IS NULL
		  )
		  AND EXISTS (
		    SELECT 1 FROM conversation_participants
		    WHERE conversation_id = c.id AND user_id = $2 AND left_at IS NULL
		  )
		LIMIT 1
	`
	c := &model.Conversation{}
	err := r.db.QueryRowContext(ctx, query, userA, userB).Scan(
		&c.ID,
		&c.Type,
		&c.Name,
		&c.CreatedBy,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: find direct between: %w", err)
	}
	return c, nil
}

func (r *conversationRepo) AddParticipant(ctx context.Context, participant *model.ConversationParticipant) error {
	query := `
		INSERT INTO conversation_participants (id, conversation_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		participant.ID,
		participant.ConversationID,
		participant.UserID,
		participant.Role,
		participant.JoinedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return model.ErrConflict
		}
		return fmt.Errorf("repo: add participant: %w", err)
	}
	return nil
}

func (r *conversationRepo) RemoveParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE conversation_participants
		SET left_at = $1
		WHERE conversation_id = $2 AND user_id = $3 AND left_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, time.Now().UTC(), conversationID, userID)
	if err != nil {
		return fmt.Errorf("repo: remove participant: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *conversationRepo) GetParticipants(ctx context.Context, conversationID uuid.UUID) ([]model.ConversationParticipant, error) {
	query := `
		SELECT id, conversation_id, user_id, role, joined_at, left_at
		FROM conversation_participants
		WHERE conversation_id = $1 AND left_at IS NULL
	`
	rows, err := r.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("repo: get participants: %w", err)
	}
	defer rows.Close()

	var participants []model.ConversationParticipant
	for rows.Next() {
		var p model.ConversationParticipant
		if err := rows.Scan(&p.ID, &p.ConversationID, &p.UserID, &p.Role, &p.JoinedAt, &p.LeftAt); err != nil {
			return nil, fmt.Errorf("repo: scan participant: %w", err)
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (r *conversationRepo) IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM conversation_participants
			WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, conversationID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repo: is participant: %w", err)
	}
	return exists, nil
}

func decodeTimeCursor(cursor string) (time.Time, uuid.UUID, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor time: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor uuid: %w", err)
	}
	return t, id, nil
}

func encodeTimeCursor(t time.Time, id uuid.UUID) string {
	raw := t.Format(time.RFC3339Nano) + "|" + id.String()
	return base64.URLEncoding.EncodeToString([]byte(raw))
}
