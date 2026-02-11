package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/repo"
)

// ConversationService handles conversation business logic.
type ConversationService struct {
	convos repo.ConversationRepository
}

// NewConversationService creates a new ConversationService.
func NewConversationService(convos repo.ConversationRepository) *ConversationService {
	return &ConversationService{convos: convos}
}

// CreateResult holds the created conversation plus whether it was existing (for direct convos).
type CreateResult struct {
	Conversation *model.Conversation
	Participants []model.ConversationParticipant
	Existing     bool
}

// Create creates a new conversation. For direct conversations, returns the existing one if it already exists.
func (s *ConversationService) Create(ctx context.Context, userID uuid.UUID, convoType string, name *string, participantIDs []uuid.UUID) (*CreateResult, error) {
	convoType = strings.TrimSpace(strings.ToLower(convoType))

	if convoType != "direct" && convoType != "group" {
		return nil, &model.ValidationError{Field: "type", Message: "must be 'direct' or 'group'"}
	}

	if convoType == "direct" {
		if len(participantIDs) != 1 {
			return nil, &model.ValidationError{Field: "participant_ids", Message: "direct conversations require exactly one other participant"}
		}
		if participantIDs[0] == userID {
			return nil, &model.ValidationError{Field: "participant_ids", Message: "cannot create a direct conversation with yourself"}
		}

		// Check for existing direct conversation.
		existing, err := s.convos.FindDirectBetween(ctx, userID, participantIDs[0])
		if err == nil {
			participants, _ := s.convos.GetParticipants(ctx, existing.ID)
			return &CreateResult{Conversation: existing, Participants: participants, Existing: true}, nil
		}
		if !errors.Is(err, model.ErrNotFound) {
			return nil, err
		}
	}

	if convoType == "group" {
		if len(participantIDs) == 0 {
			return nil, &model.ValidationError{Field: "participant_ids", Message: "group conversations require at least one other participant"}
		}
	}

	now := time.Now().UTC()
	convo := &model.Conversation{
		ID:        uuid.New(),
		Type:      convoType,
		Name:      name,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.convos.Create(ctx, convo); err != nil {
		return nil, err
	}

	// Add the creator as owner.
	ownerParticipant := &model.ConversationParticipant{
		ID:             uuid.New(),
		ConversationID: convo.ID,
		UserID:         userID,
		Role:           "owner",
		JoinedAt:       now,
	}
	if err := s.convos.AddParticipant(ctx, ownerParticipant); err != nil {
		return nil, err
	}

	allParticipants := []model.ConversationParticipant{*ownerParticipant}

	// Add the other participants as members.
	for _, pid := range participantIDs {
		p := &model.ConversationParticipant{
			ID:             uuid.New(),
			ConversationID: convo.ID,
			UserID:         pid,
			Role:           "member",
			JoinedAt:       now,
		}
		if err := s.convos.AddParticipant(ctx, p); err != nil {
			return nil, err
		}
		allParticipants = append(allParticipants, *p)
	}

	return &CreateResult{Conversation: convo, Participants: allParticipants, Existing: false}, nil
}

// List returns the authenticated user's conversations with cursor-based pagination.
func (s *ConversationService) List(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*model.Page[model.ConversationSummary], error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.convos.ListByUser(ctx, userID, cursor, limit)
}

// GetByID returns a conversation and its participants. The caller must be a participant.
func (s *ConversationService) GetByID(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*model.Conversation, []model.ConversationParticipant, error) {
	ok, err := s.convos.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, model.ErrNotFound
	}

	convo, err := s.convos.GetByID(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}

	participants, err := s.convos.GetParticipants(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}

	return convo, participants, nil
}

// Update renames a group conversation. The caller must be a participant.
func (s *ConversationService) Update(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, name string) (*model.Conversation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, &model.ValidationError{Field: "name", Message: "must not be empty"}
	}

	ok, err := s.convos.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}

	return s.convos.Update(ctx, conversationID, name)
}

// AddParticipants adds users to a group conversation. The caller must be a participant.
func (s *ConversationService) AddParticipants(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, newUserIDs []uuid.UUID) ([]model.ConversationParticipant, error) {
	if len(newUserIDs) == 0 {
		return nil, &model.ValidationError{Field: "user_ids", Message: "must provide at least one user"}
	}

	ok, err := s.convos.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}

	convo, err := s.convos.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if convo.Type != "group" {
		return nil, &model.ValidationError{Field: "type", Message: "can only add participants to group conversations"}
	}

	now := time.Now().UTC()
	for _, uid := range newUserIDs {
		p := &model.ConversationParticipant{
			ID:             uuid.New(),
			ConversationID: conversationID,
			UserID:         uid,
			Role:           "member",
			JoinedAt:       now,
		}
		if err := s.convos.AddParticipant(ctx, p); err != nil {
			if errors.Is(err, model.ErrConflict) {
				continue // already a participant, skip
			}
			return nil, err
		}
	}

	return s.convos.GetParticipants(ctx, conversationID)
}

// RemoveParticipant removes a user from a group conversation. The caller must be a participant.
func (s *ConversationService) RemoveParticipant(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, targetUserID uuid.UUID) error {
	ok, err := s.convos.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return model.ErrNotFound
	}

	convo, err := s.convos.GetByID(ctx, conversationID)
	if err != nil {
		return err
	}
	if convo.Type != "group" {
		return &model.ValidationError{Field: "type", Message: "can only remove participants from group conversations"}
	}

	return s.convos.RemoveParticipant(ctx, conversationID, targetUserID)
}
