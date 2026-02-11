package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock: UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	mu      sync.Mutex
	byEmail map[string]*model.User
	byID    map[uuid.UUID]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byEmail: make(map[string]*model.User),
		byID:    make(map[uuid.UUID]*model.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[user.Email]; exists {
		return model.ErrConflict
	}
	m.byEmail[user.Email] = user
	m.byID[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byEmail[email]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Update(_ context.Context, id uuid.UUID, params model.UpdateProfileParams) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	if params.DisplayName != nil {
		u.DisplayName = *params.DisplayName
	}
	if params.AvatarURL != nil {
		u.AvatarURL = params.AvatarURL
	}
	u.UpdatedAt = time.Now().UTC()
	return u, nil
}

func (m *mockUserRepo) Search(_ context.Context, _ string, _ string, _ int) (*model.Page[model.UserSearchResult], error) {
	return &model.Page[model.UserSearchResult]{Items: []model.UserSearchResult{}}, nil
}

// ---------------------------------------------------------------------------
// Mock: SessionRepository
// ---------------------------------------------------------------------------

type mockSessionRepo struct {
	mu       sync.Mutex
	sessions []*model.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{}
}

func (m *mockSessionRepo) Create(_ context.Context, session *model.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = append(m.sessions, session)
	return nil
}

func (m *mockSessionRepo) GetByToken(_ context.Context, tokenHash string) (*model.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.RefreshTokenHash == tokenHash && s.RevokedAt == nil {
			return s, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockSessionRepo) Revoke(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.ID == id && s.RevokedAt == nil {
			now := time.Now().UTC()
			s.RevokedAt = &now
			return nil
		}
	}
	return model.ErrNotFound
}

func (m *mockSessionRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			s.RevokedAt = &now
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAuthConfig() AuthConfig {
	return AuthConfig{
		JWTSecret:          "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

// ---------------------------------------------------------------------------
// Tests: Register
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	svc := NewAuthService(users, sessions, testAuthConfig())

	result, err := svc.Register(context.Background(), "alice@example.com", "strongpass", "Alice")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.User.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", result.User.Email)
	}
	if result.User.DisplayName != "Alice" {
		t.Errorf("expected display name Alice, got %s", result.User.DisplayName)
	}
	if result.Tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if result.Tokens.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if result.Tokens.ExpiresIn != int((15 * time.Minute).Seconds()) {
		t.Errorf("expected ExpiresIn %d, got %d", int((15*time.Minute).Seconds()), result.Tokens.ExpiresIn)
	}

	// Verify the session was stored.
	if len(sessions.sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions.sessions))
	}
}

func TestRegister_EmptyEmail(t *testing.T) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	svc := NewAuthService(users, sessions, testAuthConfig())

	_, err := svc.Register(context.Background(), "", "strongpass", "Alice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "email" {
		t.Errorf("expected field 'email', got %q", ve.Field)
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Error("expected error to unwrap to ErrValidation")
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	svc := NewAuthService(users, sessions, testAuthConfig())

	_, err := svc.Register(context.Background(), "alice@example.com", "short", "Alice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "password" {
		t.Errorf("expected field 'password', got %q", ve.Field)
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Error("expected error to unwrap to ErrValidation")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	svc := NewAuthService(users, sessions, testAuthConfig())

	// Register the first user.
	_, err := svc.Register(context.Background(), "alice@example.com", "strongpass", "Alice")
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	// Attempt duplicate registration.
	_, err = svc.Register(context.Background(), "alice@example.com", "strongpass", "Alice2")
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
	if !errors.Is(err, model.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	svc := NewAuthService(users, sessions, testAuthConfig())

	// Register a user first so there is a valid password hash.
	_, err := svc.Register(context.Background(), "bob@example.com", "correctpass", "Bob")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	result, err := svc.Login(context.Background(), "bob@example.com", "correctpass")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.User.Email != "bob@example.com" {
		t.Errorf("expected email bob@example.com, got %s", result.User.Email)
	}
	if result.Tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if result.Tokens.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	svc := NewAuthService(users, sessions, testAuthConfig())

	// Seed a user with a known password hash.
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)
	now := time.Now().UTC()
	user := &model.User{
		ID:           uuid.New(),
		Email:        "carol@example.com",
		PasswordHash: string(hash),
		DisplayName:  "Carol",
		Status:       "offline",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	users.mu.Lock()
	users.byEmail[user.Email] = user
	users.byID[user.ID] = user
	users.mu.Unlock()

	_, err := svc.Login(context.Background(), "carol@example.com", "wrongpass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLogin_NotFound(t *testing.T) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	svc := NewAuthService(users, sessions, testAuthConfig())

	_, err := svc.Login(context.Background(), "nobody@example.com", "anypass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
