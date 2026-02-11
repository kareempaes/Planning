package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
)

// ---------------------------------------------------------------------------
// Mock: ModerationRepository
// ---------------------------------------------------------------------------

type mockModerationRepo struct {
	blocked []struct{ blockerID, blockedID uuid.UUID }
	reports []*model.Report
}

func newMockModerationRepo() *mockModerationRepo {
	return &mockModerationRepo{}
}

func (m *mockModerationRepo) Block(_ context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	m.blocked = append(m.blocked, struct{ blockerID, blockedID uuid.UUID }{blockerID, blockedID})
	return nil
}

func (m *mockModerationRepo) Unblock(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockModerationRepo) ListBlocked(_ context.Context, _ uuid.UUID) ([]model.BlockedUser, error) {
	return []model.BlockedUser{}, nil
}

func (m *mockModerationRepo) IsBlocked(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockModerationRepo) CreateReport(_ context.Context, report *model.Report) error {
	m.reports = append(m.reports, report)
	return nil
}

// ---------------------------------------------------------------------------
// Tests: Block
// ---------------------------------------------------------------------------

func TestBlock_Success(t *testing.T) {
	mod := newMockModerationRepo()
	svc := NewModerationService(mod)

	blockerID := uuid.New()
	blockedID := uuid.New()

	err := svc.Block(context.Background(), blockerID, blockedID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mod.blocked) != 1 {
		t.Fatalf("expected 1 block record, got %d", len(mod.blocked))
	}
	if mod.blocked[0].blockerID != blockerID {
		t.Errorf("expected blocker %s, got %s", blockerID, mod.blocked[0].blockerID)
	}
	if mod.blocked[0].blockedID != blockedID {
		t.Errorf("expected blocked %s, got %s", blockedID, mod.blocked[0].blockedID)
	}
}

func TestBlock_Self(t *testing.T) {
	mod := newMockModerationRepo()
	svc := NewModerationService(mod)

	userID := uuid.New()

	err := svc.Block(context.Background(), userID, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "user_id" {
		t.Errorf("expected field 'user_id', got %q", ve.Field)
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Error("expected error to unwrap to ErrValidation")
	}

	// Verify no block was recorded.
	if len(mod.blocked) != 0 {
		t.Errorf("expected 0 block records, got %d", len(mod.blocked))
	}
}

// ---------------------------------------------------------------------------
// Tests: Report
// ---------------------------------------------------------------------------

func TestReport_Success(t *testing.T) {
	mod := newMockModerationRepo()
	svc := NewModerationService(mod)

	reporterID := uuid.New()
	targetID := uuid.New()

	report, err := svc.Report(context.Background(), reporterID, "user", targetID, "Spam behavior")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.ReporterID != reporterID {
		t.Errorf("expected reporter %s, got %s", reporterID, report.ReporterID)
	}
	if report.TargetType != "user" {
		t.Errorf("expected target type 'user', got %q", report.TargetType)
	}
	if report.TargetID != targetID {
		t.Errorf("expected target %s, got %s", targetID, report.TargetID)
	}
	if report.Reason != "Spam behavior" {
		t.Errorf("expected reason 'Spam behavior', got %q", report.Reason)
	}
	if report.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", report.Status)
	}

	if len(mod.reports) != 1 {
		t.Errorf("expected 1 stored report, got %d", len(mod.reports))
	}
}

func TestReport_InvalidTargetType(t *testing.T) {
	mod := newMockModerationRepo()
	svc := NewModerationService(mod)

	reporterID := uuid.New()
	targetID := uuid.New()

	_, err := svc.Report(context.Background(), reporterID, "invalid", targetID, "Some reason")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "target_type" {
		t.Errorf("expected field 'target_type', got %q", ve.Field)
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Error("expected error to unwrap to ErrValidation")
	}

	// Verify no report was stored.
	if len(mod.reports) != 0 {
		t.Errorf("expected 0 stored reports, got %d", len(mod.reports))
	}
}
