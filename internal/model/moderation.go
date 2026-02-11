package model

import (
	"time"

	"github.com/google/uuid"
)

// BlockedUser represents a blocking relationship between two users.
type BlockedUser struct {
	ID        uuid.UUID `json:"id"`
	BlockerID uuid.UUID `json:"blocker_id"`
	BlockedID uuid.UUID `json:"blocked_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Report represents a user-submitted report against a user, message, or conversation.
type Report struct {
	ID         uuid.UUID  `json:"id"`
	ReporterID uuid.UUID  `json:"reporter_id"`
	TargetType string     `json:"target_type"`
	TargetID   uuid.UUID  `json:"target_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}
