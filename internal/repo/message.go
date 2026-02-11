package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
)

// MessageRepository defines the data access contract for messages.
type MessageRepository interface {
	Create(ctx context.Context, msg *model.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error)
	ListByConversation(ctx context.Context, conversationID uuid.UUID, cursor string, limit int) (*model.Page[model.Message], error)
	CreateDeliveries(ctx context.Context, messageID uuid.UUID, userIDs []uuid.UUID) error
	UpdateDeliveryStatus(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, status string) error
}

type messageRepo struct {
	db *sql.DB
}

// NewMessageRepo creates a new MessageRepository backed by the given database.
func NewMessageRepo(db *sql.DB) MessageRepository {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, msg *model.Message) error {
	query := `
		INSERT INTO messages (id, conversation_id, sender_id, body, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		msg.ID,
		msg.ConversationID,
		msg.SenderID,
		msg.Body,
		msg.Status,
		msg.CreatedAt,
		msg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repo: create message: %w", err)
	}
	return nil
}

func (r *messageRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, body, status, created_at, updated_at
		FROM messages
		WHERE id = $1
	`
	m := &model.Message{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID,
		&m.ConversationID,
		&m.SenderID,
		&m.Body,
		&m.Status,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repo: get message by id: %w", err)
	}
	return m, nil
}

func (r *messageRepo) ListByConversation(ctx context.Context, conversationID uuid.UUID, cursor string, limit int) (*model.Page[model.Message], error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	fetchLimit := limit + 1

	args := []any{conversationID}
	argIdx := 2

	whereCursor := ""
	if cursor != "" {
		cursorTime, cursorID, err := decodeTimeCursor(cursor)
		if err != nil {
			return nil, &model.ValidationError{Field: "cursor", Message: "invalid cursor"}
		}
		whereCursor = fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, cursorTime, cursorID)
		argIdx += 2
	}

	args = append(args, fetchLimit)

	query := fmt.Sprintf(`
		SELECT id, conversation_id, sender_id, body, status, created_at, updated_at
		FROM messages
		WHERE conversation_id = $1%s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d
	`, whereCursor, argIdx)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repo: list messages: %w", err)
	}
	defer rows.Close()

	results := make([]model.Message, 0, limit)
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repo: scan message: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo: list messages rows error: %w", err)
	}

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	var nextCursor *string
	if hasMore && len(results) > 0 {
		last := results[len(results)-1]
		c := encodeTimeCursor(last.CreatedAt, last.ID)
		nextCursor = &c
	}

	return &model.Page[model.Message]{
		Items:      results,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *messageRepo) CreateDeliveries(ctx context.Context, messageID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(userIDs))
	args := make([]any, 0, len(userIDs)*2)
	argIdx := 1

	for _, uid := range userIDs {
		valueStrings = append(valueStrings, fmt.Sprintf("(gen_random_uuid(), $%d, $%d)", argIdx, argIdx+1))
		args = append(args, messageID, uid)
		argIdx += 2
	}

	query := fmt.Sprintf(`
		INSERT INTO message_deliveries (id, message_id, user_id)
		VALUES %s
	`, strings.Join(valueStrings, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("repo: create deliveries: %w", err)
	}
	return nil
}

func (r *messageRepo) UpdateDeliveryStatus(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, status string) error {
	query := `
		UPDATE message_deliveries
		SET status = $1, delivered_at = $2
		WHERE message_id = $3 AND user_id = $4
	`
	res, err := r.db.ExecContext(ctx, query, status, time.Now().UTC(), messageID, userID)
	if err != nil {
		return fmt.Errorf("repo: update delivery status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}
