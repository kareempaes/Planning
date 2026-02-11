package dto

import (
	"time"

	"github.com/google/uuid"
)

// SendMessageRequest is the body for POST /conversations/:id/messages.
type SendMessageRequest struct {
	Body string `json:"body"`
}

// MessageResponse is a single message in an API response.
type MessageResponse struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Body           string    `json:"body"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// MessageListResponse is the response for GET /conversations/:id/messages.
type MessageListResponse struct {
	Messages   []MessageResponse  `json:"messages"`
	Pagination PaginationResponse `json:"pagination"`
}
