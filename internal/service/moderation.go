package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/repo"
)

// ModerationService handles moderation business logic.
type ModerationService struct {
	mod repo.ModerationRepository
}

// NewModerationService creates a new ModerationService.
func NewModerationService(mod repo.ModerationRepository) *ModerationService {
	return &ModerationService{mod: mod}
}

// Block blocks a user. The caller cannot block themselves.
func (s *ModerationService) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return &model.ValidationError{Field: "user_id", Message: "cannot block yourself"}
	}
	return s.mod.Block(ctx, blockerID, blockedID)
}

// Unblock removes a block on a user.
func (s *ModerationService) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	return s.mod.Unblock(ctx, blockerID, blockedID)
}

// ListBlocked returns all users blocked by the given user.
func (s *ModerationService) ListBlocked(ctx context.Context, blockerID uuid.UUID) ([]model.BlockedUser, error) {
	return s.mod.ListBlocked(ctx, blockerID)
}

// Report creates a new report against a user, message, or conversation.
func (s *ModerationService) Report(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*model.Report, error) {
	targetType = strings.TrimSpace(strings.ToLower(targetType))
	reason = strings.TrimSpace(reason)

	if targetType != "user" && targetType != "message" && targetType != "conversation" {
		return nil, &model.ValidationError{Field: "target_type", Message: "must be 'user', 'message', or 'conversation'"}
	}
	if reason == "" {
		return nil, &model.ValidationError{Field: "reason", Message: "must not be empty"}
	}
	if len(reason) > 1000 {
		return nil, &model.ValidationError{Field: "reason", Message: "must be 1000 characters or fewer"}
	}

	now := time.Now().UTC()
	report := &model.Report{
		ID:         uuid.New(),
		ReporterID: reporterID,
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     reason,
		Status:     "pending",
		CreatedAt:  now,
	}

	if err := s.mod.CreateReport(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}
