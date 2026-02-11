package model

import (
	"time"

	"github.com/google/uuid"
)

// Conversation represents a direct or group conversation.
type Conversation struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Name      *string   `json:"name"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ConversationParticipant represents a user's membership in a conversation.
type ConversationParticipant struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	UserID         uuid.UUID  `json:"user_id"`
	Role           string     `json:"role"`
	JoinedAt       time.Time  `json:"joined_at"`
	LeftAt         *time.Time `json:"left_at"`
}

// ConversationSummary is the list-view representation of a conversation.
type ConversationSummary struct {
	ID           uuid.UUID            `json:"id"`
	Type         string               `json:"type"`
	Name         *string              `json:"name"`
	LastMessage  *MessagePreview      `json:"last_message"`
	UnreadCount  int                  `json:"unread_count"`
	Participants []ParticipantSummary `json:"participants"`
}

// ParticipantSummary is a lightweight participant reference.
type ParticipantSummary struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
}

// MessagePreview is a compact message representation for conversation lists.
type MessagePreview struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	SenderID  uuid.UUID `json:"sender_id"`
	CreatedAt time.Time `json:"created_at"`
}
