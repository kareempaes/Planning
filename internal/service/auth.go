package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// AuthConfig holds configuration for the authentication service.
type AuthConfig struct {
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// AuthTokens is the token pair returned to the client.
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// AuthResult combines a user with their issued tokens.
type AuthResult struct {
	User   *model.User
	Tokens AuthTokens
}

// AuthService handles authentication business logic.
type AuthService struct {
	users    repo.UserRepository
	sessions repo.SessionRepository
	config   AuthConfig
}

// NewAuthService creates a new AuthService.
func NewAuthService(users repo.UserRepository, sessions repo.SessionRepository, config AuthConfig) *AuthService {
	if config.AccessTokenExpiry == 0 {
		config.AccessTokenExpiry = 15 * time.Minute
	}
	if config.RefreshTokenExpiry == 0 {
		config.RefreshTokenExpiry = 7 * 24 * time.Hour
	}
	return &AuthService{users: users, sessions: sessions, config: config}
}

// Register creates a new user account and returns tokens.
func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*AuthResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	displayName = strings.TrimSpace(displayName)

	if email == "" {
		return nil, &model.ValidationError{Field: "email", Message: "must not be empty"}
	}
	if password == "" {
		return nil, &model.ValidationError{Field: "password", Message: "must not be empty"}
	}
	if len(password) < 8 {
		return nil, &model.ValidationError{Field: "password", Message: "must be at least 8 characters"}
	}
	if displayName == "" {
		return nil, &model.ValidationError{Field: "display_name", Message: "must not be empty"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &model.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Status:       "offline",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	tokens, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: user, Tokens: *tokens}, nil
}

// Login authenticates an existing user and returns tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return nil, &model.ValidationError{Field: "email", Message: "must not be empty"}
	}
	if password == "" {
		return nil, &model.ValidationError{Field: "password", Message: "must not be empty"}
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, model.ErrNotFound
	}

	tokens, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: user, Tokens: *tokens}, nil
}

// RefreshToken validates a refresh token and issues a new token pair.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	if refreshToken == "" {
		return nil, &model.ValidationError{Field: "refresh_token", Message: "must not be empty"}
	}

	tokenHash := hashToken(refreshToken)

	session, err := s.sessions.GetByToken(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	// Revoke the old session (rotate refresh tokens).
	_ = s.sessions.Revoke(ctx, session.ID)

	return s.issueTokens(ctx, session.UserID)
}

// Logout revokes the session associated with the given refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return &model.ValidationError{Field: "refresh_token", Message: "must not be empty"}
	}

	tokenHash := hashToken(refreshToken)

	session, err := s.sessions.GetByToken(ctx, tokenHash)
	if err != nil {
		return err
	}

	return s.sessions.Revoke(ctx, session.ID)
}

// issueTokens generates a JWT access token and a random refresh token, persisting the session.
func (s *AuthService) issueTokens(ctx context.Context, userID uuid.UUID) (*AuthTokens, error) {
	now := time.Now().UTC()

	// Generate access token (JWT).
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": now.Unix(),
		"exp": now.Add(s.config.AccessTokenExpiry).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("auth: sign access token: %w", err)
	}

	// Generate refresh token (random).
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("auth: generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(refreshBytes)

	// Persist session with hashed refresh token.
	session := &model.Session{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshTokenHash: hashToken(refreshToken),
		ExpiresAt:        now.Add(s.config.RefreshTokenExpiry),
		CreatedAt:        now,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

// hashToken returns the hex-encoded SHA-256 hash of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
