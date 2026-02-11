package model

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message within a conversation.
type Message struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Body           string    `json:"body"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MessageDelivery tracks per-user delivery status of a message.
type MessageDelivery struct {
	ID          uuid.UUID  `json:"id"`
	MessageID   uuid.UUID  `json:"message_id"`
	UserID      uuid.UUID  `json:"user_id"`
	Status      string     `json:"status"`
	DeliveredAt *time.Time `json:"delivered_at"`
	ReadAt      *time.Time `json:"read_at"`
}
