package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/dto"
	"github.com/kareempaes/planning/internal/service"
)

// ModerationHandler handles moderation endpoints.
type ModerationHandler struct {
	mod   *service.ModerationService
	users *service.UserService
}

// NewModerationHandler creates a new ModerationHandler.
func NewModerationHandler(mod *service.ModerationService, users *service.UserService) *ModerationHandler {
	return &ModerationHandler{mod: mod, users: users}
}

// Block handles POST /users/{id}/block.
func (h *ModerationHandler) Block(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	blockedID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid user ID"},
		})
		return
	}

	if err := h.mod.Block(r.Context(), userID, blockedID); err != nil {
		writeError(w, err)
		return
	}

	writeNoContent(w)
}

// Unblock handles DELETE /users/{id}/block.
func (h *ModerationHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	blockedID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid user ID"},
		})
		return
	}

	if err := h.mod.Unblock(r.Context(), userID, blockedID); err != nil {
		writeError(w, err)
		return
	}

	writeNoContent(w)
}

// ListBlocked handles GET /users/me/blocked.
func (h *ModerationHandler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	blocked, err := h.mod.ListBlocked(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}

	items := make([]dto.BlockedUserDTO, 0, len(blocked))
	for _, b := range blocked {
		profile, err := h.users.GetPublicProfile(r.Context(), b.BlockedID)
		if err != nil {
			continue // user may have been deleted
		}
		items = append(items, dto.BlockedUserDTO{
			UserID:      b.BlockedID,
			DisplayName: profile.DisplayName,
			BlockedAt:   b.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, dto.BlockedListResponse{Blocked: items})
}

// Report handles POST /reports.
func (h *ModerationHandler) Report(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req dto.ReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid target ID"},
		})
		return
	}

	report, err := h.mod.Report(r.Context(), userID, req.TargetType, targetID, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ReportResponse{
		ID:         report.ID,
		TargetType: report.TargetType,
		TargetID:   report.TargetID,
		Status:     report.Status,
		CreatedAt:  report.CreatedAt,
	})
}
