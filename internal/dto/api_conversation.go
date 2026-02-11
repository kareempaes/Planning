package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateConversationRequest is the body for POST /conversations.
type CreateConversationRequest struct {
	Type           string   `json:"type"`
	ParticipantIDs []string `json:"participant_ids"`
	Name           *string  `json:"name"`
}

// UpdateConversationRequest is the body for PATCH /conversations/:id.
type UpdateConversationRequest struct {
	Name string `json:"name"`
}

// AddParticipantsRequest is the body for POST /conversations/:id/participants.
type AddParticipantsRequest struct {
	UserIDs []string `json:"user_ids"`
}

// ParticipantResponse is a participant in a conversation response.
type ParticipantResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
}

// ConversationResponse is the full conversation detail response.
type ConversationResponse struct {
	ID           uuid.UUID             `json:"id"`
	Type         string                `json:"type"`
	Name         *string               `json:"name"`
	Participants []ParticipantResponse `json:"participants"`
	CreatedAt    time.Time             `json:"created_at"`
}

// ConversationListResponse is the response for GET /conversations.
type ConversationListResponse struct {
	Conversations []ConversationSummaryResponse `json:"conversations"`
	Pagination    PaginationResponse            `json:"pagination"`
}

// ConversationSummaryResponse is a lightweight conversation for list views.
type ConversationSummaryResponse struct {
	ID           uuid.UUID             `json:"id"`
	Type         string                `json:"type"`
	Name         *string               `json:"name"`
	LastMessage  *MessagePreviewDTO    `json:"last_message"`
	UnreadCount  int                   `json:"unread_count"`
	Participants []ParticipantMinDTO   `json:"participants"`
}

// MessagePreviewDTO is a compact message for conversation list previews.
type MessagePreviewDTO struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	SenderID  uuid.UUID `json:"sender_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ParticipantMinDTO is a minimal participant reference for list views.
type ParticipantMinDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
}
