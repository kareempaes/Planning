package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/dto"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/service"
)

// UserHandler handles user endpoints.
type UserHandler struct {
	users *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

// GetMe handles GET /users/me.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	user, err := h.users.GetProfile(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// UpdateMe handles PATCH /users/me.
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req dto.UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	user, err := h.users.UpdateProfile(r.Context(), userID, model.UpdateProfileParams{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// GetPublicProfile handles GET /users/{id}.
func (h *UserHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid user ID"},
		})
		return
	}

	profile, err := h.users.GetPublicProfile(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.PublicProfileResponse{
		ID:          profile.ID,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Status:      profile.Status,
	})
}

// Search handles GET /users?q=&cursor=&limit=.
func (h *UserHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	cursor := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	page, err := h.users.SearchUsers(r.Context(), q, cursor, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	users := make([]dto.PublicProfileResponse, len(page.Items))
	for i, u := range page.Items {
		users[i] = dto.PublicProfileResponse{
			ID:          u.ID,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
		}
	}

	writeJSON(w, http.StatusOK, dto.SearchResponse{
		Users: users,
		Pagination: dto.PaginationResponse{
			NextCursor: page.NextCursor,
			HasMore:    page.HasMore,
		},
	})
}
