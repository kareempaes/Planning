package dto

import (
	"time"

	"github.com/google/uuid"
)

// ReportRequest is the body for POST /reports.
type ReportRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Reason     string `json:"reason"`
}

// ReportResponse is the response for a created report.
type ReportResponse struct {
	ID         uuid.UUID `json:"id"`
	TargetType string    `json:"target_type"`
	TargetID   uuid.UUID `json:"target_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// BlockedUserDTO represents a blocked user in the blocked list response.
type BlockedUserDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	BlockedAt   time.Time `json:"blocked_at"`
}

// BlockedListResponse is the response for GET /users/me/blocked.
type BlockedListResponse struct {
	Blocked []BlockedUserDTO `json:"blocked"`
}
