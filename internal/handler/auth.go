package handler

import (
	"errors"
	"net/http"

	"github.com/kareempaes/planning/internal/dto"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	result, err := h.auth.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.AuthResponse{
		User:   toUserResponse(result.User),
		Tokens: toTokenResponse(&result.Tokens),
	})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	result, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeJSON(w, http.StatusUnauthorized, ErrorBody{
				Error: ErrorDetail{Code: "unauthorized", Message: "invalid credentials"},
			})
			return
		}
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.AuthResponse{
		User:   toUserResponse(result.User),
		Tokens: toTokenResponse(&result.Tokens),
	})
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	tokens, err := h.auth.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toTokenResponse(tokens))
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	if err := h.auth.Logout(r.Context(), req.RefreshToken); err != nil {
		writeError(w, err)
		return
	}

	writeNoContent(w)
}

func toUserResponse(u *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
	}
}

func toTokenResponse(t *service.AuthTokens) dto.TokenResponse {
	return dto.TokenResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresIn:    t.ExpiresIn,
	}
}
