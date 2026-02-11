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
// Mock: ConversationRepository
// ---------------------------------------------------------------------------

type mockConversationRepo struct {
	mu           sync.Mutex
	conversations map[uuid.UUID]*model.Conversation
	participants  map[uuid.UUID][]model.ConversationParticipant // keyed by conversation ID
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		conversations: make(map[uuid.UUID]*model.Conversation),
		participants:  make(map[uuid.UUID][]model.ConversationParticipant),
	}
}

func (m *mockConversationRepo) Create(_ context.Context, convo *model.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conversations[convo.ID] = convo
	return nil
}

func (m *mockConversationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.conversations[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return c, nil
}

func (m *mockConversationRepo) ListByUser(_ context.Context, _ uuid.UUID, _ string, _ int) (*model.Page[model.ConversationSummary], error) {
	return &model.Page[model.ConversationSummary]{Items: []model.ConversationSummary{}}, nil
}

func (m *mockConversationRepo) Update(_ context.Context, id uuid.UUID, name string) (*model.Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.conversations[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	c.Name = &name
	return c, nil
}

func (m *mockConversationRepo) FindDirectBetween(_ context.Context, userA uuid.UUID, userB uuid.UUID) (*model.Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.conversations {
		if c.Type != "direct" {
			continue
		}
		parts := m.participants[c.ID]
		hasA, hasB := false, false
		for _, p := range parts {
			if p.UserID == userA {
				hasA = true
			}
			if p.UserID == userB {
				hasB = true
			}
		}
		if hasA && hasB {
			return c, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockConversationRepo) AddParticipant(_ context.Context, participant *model.ConversationParticipant) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.participants[participant.ConversationID] = append(m.participants[participant.ConversationID], *participant)
	return nil
}

func (m *mockConversationRepo) RemoveParticipant(_ context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	parts, ok := m.participants[conversationID]
	if !ok {
		return model.ErrNotFound
	}
	for i, p := range parts {
		if p.UserID == userID {
			m.participants[conversationID] = append(parts[:i], parts[i+1:]...)
			return nil
		}
	}
	return model.ErrNotFound
}

func (m *mockConversationRepo) GetParticipants(_ context.Context, conversationID uuid.UUID) ([]model.ConversationParticipant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.participants[conversationID], nil
}

func (m *mockConversationRepo) IsParticipant(_ context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.participants[conversationID] {
		if p.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// Tests: Create
// ---------------------------------------------------------------------------

func TestCreate_Direct(t *testing.T) {
	convos := newMockConversationRepo()
	svc := NewConversationService(convos)

	userID := uuid.New()
	otherID := uuid.New()

	result, err := svc.Create(context.Background(), userID, "direct", nil, []uuid.UUID{otherID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Conversation.Type != "direct" {
		t.Errorf("expected type 'direct', got %q", result.Conversation.Type)
	}
	if result.Existing {
		t.Error("expected Existing to be false for a new conversation")
	}
	if len(result.Participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(result.Participants))
	}

	// Verify participant roles.
	foundOwner := false
	foundMember := false
	for _, p := range result.Participants {
		if p.UserID == userID && p.Role == "owner" {
			foundOwner = true
		}
		if p.UserID == otherID && p.Role == "member" {
			foundMember = true
		}
	}
	if !foundOwner {
		t.Error("expected creator to have 'owner' role")
	}
	if !foundMember {
		t.Error("expected other participant to have 'member' role")
	}
}

func TestCreate_DirectWithSelf(t *testing.T) {
	convos := newMockConversationRepo()
	svc := NewConversationService(convos)

	userID := uuid.New()

	_, err := svc.Create(context.Background(), userID, "direct", nil, []uuid.UUID{userID})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "participant_ids" {
		t.Errorf("expected field 'participant_ids', got %q", ve.Field)
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Error("expected error to unwrap to ErrValidation")
	}
}

func TestCreate_Group(t *testing.T) {
	convos := newMockConversationRepo()
	svc := NewConversationService(convos)

	userID := uuid.New()
	member1 := uuid.New()
	member2 := uuid.New()
	groupName := "Test Group"

	result, err := svc.Create(context.Background(), userID, "group", &groupName, []uuid.UUID{member1, member2})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Conversation.Type != "group" {
		t.Errorf("expected type 'group', got %q", result.Conversation.Type)
	}
	if result.Conversation.Name == nil || *result.Conversation.Name != groupName {
		t.Errorf("expected name %q, got %v", groupName, result.Conversation.Name)
	}
	if result.Existing {
		t.Error("expected Existing to be false")
	}
	// 1 owner + 2 members = 3 participants.
	if len(result.Participants) != 3 {
		t.Errorf("expected 3 participants, got %d", len(result.Participants))
	}
}

func TestCreate_InvalidType(t *testing.T) {
	convos := newMockConversationRepo()
	svc := NewConversationService(convos)

	userID := uuid.New()

	_, err := svc.Create(context.Background(), userID, "invalid", nil, []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "type" {
		t.Errorf("expected field 'type', got %q", ve.Field)
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Error("expected error to unwrap to ErrValidation")
	}
}
