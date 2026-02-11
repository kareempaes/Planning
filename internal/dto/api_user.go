package dto

import "github.com/google/uuid"

// UpdateProfileRequest is the body for PATCH /users/me.
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// PublicProfileResponse is a user's public-facing profile.
type PublicProfileResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	Status      string    `json:"status"`
}

// SearchResponse is the response for GET /users?q=.
type SearchResponse struct {
	Users      []PublicProfileResponse `json:"users"`
	Pagination PaginationResponse     `json:"pagination"`
}

// PaginationResponse is a shared cursor-pagination envelope.
type PaginationResponse struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}
