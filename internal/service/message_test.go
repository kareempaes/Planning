package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
)

// ---------------------------------------------------------------------------
// Mock: MessageRepository
// ---------------------------------------------------------------------------

type mockMessageRepo struct {
	mu       sync.Mutex
	messages map[uuid.UUID]*model.Message
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		messages: make(map[uuid.UUID]*model.Message),
	}
}

func (m *mockMessageRepo) Create(_ context.Context, msg *model.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return msg, nil
}

func (m *mockMessageRepo) ListByConversation(_ context.Context, _ uuid.UUID, _ string, _ int) (*model.Page[model.Message], error) {
	return &model.Page[model.Message]{Items: []model.Message{}}, nil
}

func (m *mockMessageRepo) CreateDeliveries(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (m *mockMessageRepo) UpdateDeliveryStatus(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}

// ---------------------------------------------------------------------------
// Helper: set up a conversation with participants using mockConversationRepo
// ---------------------------------------------------------------------------

func setupConvoWithParticipants(convoRepo *mockConversationRepo, convoID uuid.UUID, convoType string, participantIDs ...uuid.UUID) {
	convoRepo.mu.Lock()
	defer convoRepo.mu.Unlock()

	convoRepo.conversations[convoID] = &model.Conversation{
		ID:   convoID,
		Type: convoType,
	}
	for _, uid := range participantIDs {
		convoRepo.participants[convoID] = append(convoRepo.participants[convoID], model.ConversationParticipant{
			ID:             uuid.New(),
			ConversationID: convoID,
			UserID:         uid,
			Role:           "member",
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: Send
// ---------------------------------------------------------------------------

func TestSend_Success(t *testing.T) {
	msgRepo := newMockMessageRepo()
	convoRepo := newMockConversationRepo()
	svc := NewMessageService(msgRepo, convoRepo)

	senderID := uuid.New()
	otherID := uuid.New()
	convoID := uuid.New()

	setupConvoWithParticipants(convoRepo, convoID, "direct", senderID, otherID)

	msg, err := svc.Send(context.Background(), senderID, convoID, "Hello!")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Body != "Hello!" {
		t.Errorf("expected body 'Hello!', got %q", msg.Body)
	}
	if msg.SenderID != senderID {
		t.Errorf("expected sender %s, got %s", senderID, msg.SenderID)
	}
	if msg.ConversationID != convoID {
		t.Errorf("expected conversation %s, got %s", convoID, msg.ConversationID)
	}
	if msg.Status != "sent" {
		t.Errorf("expected status 'sent', got %q", msg.Status)
	}

	// Verify message was persisted in the mock.
	if len(msgRepo.messages) != 1 {
		t.Errorf("expected 1 stored message, got %d", len(msgRepo.messages))
	}
}

func TestSend_NotParticipant(t *testing.T) {
	msgRepo := newMockMessageRepo()
	convoRepo := newMockConversationRepo()
	svc := NewMessageService(msgRepo, convoRepo)

	outsiderID := uuid.New()
	convoID := uuid.New()

	// Create a conversation that does not include outsiderID.
	setupConvoWithParticipants(convoRepo, convoID, "direct", uuid.New(), uuid.New())

	_, err := svc.Send(context.Background(), outsiderID, convoID, "Should fail")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSend_EmptyBody(t *testing.T) {
	msgRepo := newMockMessageRepo()
	convoRepo := newMockConversationRepo()
	svc := NewMessageService(msgRepo, convoRepo)

	senderID := uuid.New()
	convoID := uuid.New()

	setupConvoWithParticipants(convoRepo, convoID, "direct", senderID, uuid.New())

	_, err := svc.Send(context.Background(), senderID, convoID, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "body" {
		t.Errorf("expected field 'body', got %q", ve.Field)
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Error("expected error to unwrap to ErrValidation")
	}
}
