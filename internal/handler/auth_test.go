package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/dto"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/service"
)

// ---------------------------------------------------------------------------
// Mock UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	mu    sync.Mutex
	users map[string]*model.User // keyed by email
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.users[user.Email]; exists {
		return model.ErrConflict
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[email]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Update(_ context.Context, id uuid.UUID, params model.UpdateProfileParams) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			if params.DisplayName != nil {
				u.DisplayName = *params.DisplayName
			}
			if params.AvatarURL != nil {
				u.AvatarURL = params.AvatarURL
			}
			u.UpdatedAt = time.Now().UTC()
			return u, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockUserRepo) Search(_ context.Context, _ string, _ string, _ int) (*model.Page[model.UserSearchResult], error) {
	return &model.Page[model.UserSearchResult]{Items: nil, HasMore: false}, nil
}

// ---------------------------------------------------------------------------
// Mock SessionRepository
// ---------------------------------------------------------------------------

type mockSessionRepo struct {
	mu       sync.Mutex
	sessions []*model.Session
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

func newTestAuthHandler() *AuthHandler {
	mockUsers := &mockUserRepo{users: make(map[string]*model.User)}
	mockSessions := &mockSessionRepo{}

	authSvc := service.NewAuthService(mockUsers, mockSessions, service.AuthConfig{
		JWTSecret:          "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	return NewAuthHandler(authSvc)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRegisterHandler_Success(t *testing.T) {
	h := newTestAuthHandler()

	body := `{"email":"alice@example.com","password":"securepass","display_name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp dto.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.User.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", resp.User.Email)
	}
	if resp.User.DisplayName != "Alice" {
		t.Errorf("expected display_name Alice, got %s", resp.User.DisplayName)
	}
	if resp.Tokens.AccessToken == "" {
		t.Errorf("expected non-empty access token")
	}
	if resp.Tokens.RefreshToken == "" {
		t.Errorf("expected non-empty refresh token")
	}
}

func TestRegisterHandler_BadJSON(t *testing.T) {
	h := newTestAuthHandler()

	body := `{not valid json}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestLoginHandler_Success(t *testing.T) {
	h := newTestAuthHandler()

	// Register a user first.
	regBody := `{"email":"bob@example.com","password":"securepass","display_name":"Bob"}`
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	h.Register(regRec, regReq)

	if regRec.Code != http.StatusCreated {
		t.Fatalf("register failed with status %d: %s", regRec.Code, regRec.Body.String())
	}

	// Login with the same credentials.
	loginBody := `{"email":"bob@example.com","password":"securepass"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", loginRec.Code, loginRec.Body.String())
	}

	var resp dto.AuthResponse
	if err := json.Unmarshal(loginRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.User.Email != "bob@example.com" {
		t.Errorf("expected email bob@example.com, got %s", resp.User.Email)
	}
	if resp.Tokens.AccessToken == "" {
		t.Errorf("expected non-empty access token")
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	h := newTestAuthHandler()

	// Register a user first.
	regBody := `{"email":"carol@example.com","password":"correctpass","display_name":"Carol"}`
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	h.Register(regRec, regReq)

	if regRec.Code != http.StatusCreated {
		t.Fatalf("register failed with status %d: %s", regRec.Code, regRec.Body.String())
	}

	// Login with the wrong password.
	loginBody := `{"email":"carol@example.com","password":"wrongpass1"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d; body: %s", loginRec.Code, loginRec.Body.String())
	}
}
