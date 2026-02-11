package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/repo"
)

// MessageService handles message business logic.
type MessageService struct {
	messages repo.MessageRepository
	convos   repo.ConversationRepository
}

// NewMessageService creates a new MessageService.
func NewMessageService(messages repo.MessageRepository, convos repo.ConversationRepository) *MessageService {
	return &MessageService{messages: messages, convos: convos}
}

// Send creates a new message in a conversation. The caller must be a participant.
func (s *MessageService) Send(ctx context.Context, senderID uuid.UUID, conversationID uuid.UUID, body string) (*model.Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, &model.ValidationError{Field: "body", Message: "must not be empty"}
	}
	if len(body) > 10000 {
		return nil, &model.ValidationError{Field: "body", Message: "must be 10000 characters or fewer"}
	}

	ok, err := s.convos.IsParticipant(ctx, conversationID, senderID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}

	now := time.Now().UTC()
	msg := &model.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Body:           body,
		Status:         "sent",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, err
	}

	// Create delivery records for all participants except the sender.
	participants, err := s.convos.GetParticipants(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	recipientIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		if p.UserID != senderID {
			recipientIDs = append(recipientIDs, p.UserID)
		}
	}

	if len(recipientIDs) > 0 {
		if err := s.messages.CreateDeliveries(ctx, msg.ID, recipientIDs); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// GetHistory returns paginated messages for a conversation. The caller must be a participant.
func (s *MessageService) GetHistory(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, cursor string, limit int) (*model.Page[model.Message], error) {
	ok, err := s.convos.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	return s.messages.ListByConversation(ctx, conversationID, cursor, limit)
}

// GetByID returns a single message. The caller must be a participant in the conversation.
func (s *MessageService) GetByID(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) (*model.Message, error) {
	ok, err := s.convos.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}

	msg, err := s.messages.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Verify the message belongs to the requested conversation.
	if msg.ConversationID != conversationID {
		return nil, model.ErrNotFound
	}

	return msg, nil
}
