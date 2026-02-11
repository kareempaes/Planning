package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/repo"
)

// UserService handles user profile business logic.
type UserService struct {
	repo repo.UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(r repo.UserRepository) *UserService {
	return &UserService{repo: r}
}

// GetProfile returns the full profile of the authenticated user.
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.repo.GetByID(ctx, userID)
}

// UpdateProfile validates and applies profile updates for the authenticated user.
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, params model.UpdateProfileParams) (*model.User, error) {
	if params.DisplayName != nil {
		name := strings.TrimSpace(*params.DisplayName)
		if name == "" {
			return nil, &model.ValidationError{Field: "display_name", Message: "must not be empty"}
		}
		if len(name) > 100 {
			return nil, &model.ValidationError{Field: "display_name", Message: "must be 100 characters or fewer"}
		}
		params.DisplayName = &name
	}

	if params.AvatarURL != nil {
		url := strings.TrimSpace(*params.AvatarURL)
		if len(url) > 2048 {
			return nil, &model.ValidationError{Field: "avatar_url", Message: "must be 2048 characters or fewer"}
		}
		params.AvatarURL = &url
	}

	if params.DisplayName == nil && params.AvatarURL == nil {
		return nil, &model.ValidationError{Field: "body", Message: "at least one field must be provided"}
	}

	return s.repo.Update(ctx, userID, params)
}

// GetPublicProfile returns the public-facing profile of any user.
func (s *UserService) GetPublicProfile(ctx context.Context, userID uuid.UUID) (*model.PublicProfile, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &model.PublicProfile{
		ID:          user.ID,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Status:      user.Status,
	}, nil
}

// SearchUsers searches for users by display name with cursor-based pagination.
func (s *UserService) SearchUsers(ctx context.Context, query string, cursor string, limit int) (*model.Page[model.UserSearchResult], error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, &model.ValidationError{Field: "q", Message: "search query must not be empty"}
	}
	if len(query) > 100 {
		return nil, &model.ValidationError{Field: "q", Message: "search query too long"}
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.repo.Search(ctx, query, cursor, limit)
}
