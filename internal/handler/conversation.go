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

// ConversationHandler handles conversation endpoints.
type ConversationHandler struct {
	convos *service.ConversationService
}

// NewConversationHandler creates a new ConversationHandler.
func NewConversationHandler(convos *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{convos: convos}
}

// Create handles POST /conversations.
func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req dto.CreateConversationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	participantIDs, err := parseUUIDs(req.ParticipantIDs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid participant ID"},
		})
		return
	}

	result, err := h.convos.Create(r.Context(), userID, req.Type, req.Name, participantIDs)
	if err != nil {
		writeError(w, err)
		return
	}

	status := http.StatusCreated
	if result.Existing {
		status = http.StatusOK
	}

	writeJSON(w, status, toConversationResponse(result.Conversation, result.Participants))
}

// List handles GET /conversations.
func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	cursor := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	page, err := h.convos.List(r.Context(), userID, cursor, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	summaries := make([]dto.ConversationSummaryResponse, len(page.Items))
	for i, s := range page.Items {
		summary := dto.ConversationSummaryResponse{
			ID:          s.ID,
			Type:        s.Type,
			Name:        s.Name,
			UnreadCount: s.UnreadCount,
		}
		if s.LastMessage != nil {
			summary.LastMessage = &dto.MessagePreviewDTO{
				ID:        s.LastMessage.ID,
				Body:      s.LastMessage.Body,
				SenderID:  s.LastMessage.SenderID,
				CreatedAt: s.LastMessage.CreatedAt,
			}
		}
		for _, p := range s.Participants {
			summary.Participants = append(summary.Participants, dto.ParticipantMinDTO{
				UserID:      p.UserID,
				DisplayName: p.DisplayName,
			})
		}
		summaries[i] = summary
	}

	writeJSON(w, http.StatusOK, dto.ConversationListResponse{
		Conversations: summaries,
		Pagination: dto.PaginationResponse{
			NextCursor: page.NextCursor,
			HasMore:    page.HasMore,
		},
	})
}

// GetByID handles GET /conversations/{id}.
func (h *ConversationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid conversation ID"},
		})
		return
	}

	convo, participants, err := h.convos.GetByID(r.Context(), userID, convoID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toConversationResponse(convo, participants))
}

// Update handles PATCH /conversations/{id}.
func (h *ConversationHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid conversation ID"},
		})
		return
	}

	var req dto.UpdateConversationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	convo, err := h.convos.Update(r.Context(), userID, convoID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ConversationResponse{
		ID:        convo.ID,
		Type:      convo.Type,
		Name:      convo.Name,
		CreatedAt: convo.CreatedAt,
	})
}

// AddParticipants handles POST /conversations/{id}/participants.
func (h *ConversationHandler) AddParticipants(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid conversation ID"},
		})
		return
	}

	var req dto.AddParticipantsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	newUserIDs, err := parseUUIDs(req.UserIDs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid user ID"},
		})
		return
	}

	participants, err := h.convos.AddParticipants(r.Context(), userID, convoID, newUserIDs)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := make([]dto.ParticipantResponse, len(participants))
	for i, p := range participants {
		resp[i] = dto.ParticipantResponse{
			UserID: p.UserID,
			Role:   p.Role,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// RemoveParticipant handles DELETE /conversations/{id}/participants/{userId}.
func (h *ConversationHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid conversation ID"},
		})
		return
	}

	targetUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid user ID"},
		})
		return
	}

	if err := h.convos.RemoveParticipant(r.Context(), userID, convoID, targetUserID); err != nil {
		writeError(w, err)
		return
	}

	writeNoContent(w)
}

func toConversationResponse(c *model.Conversation, participants []model.ConversationParticipant) dto.ConversationResponse {
	resp := dto.ConversationResponse{
		ID:        c.ID,
		Type:      c.Type,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
	}
	for _, p := range participants {
		resp.Participants = append(resp.Participants, dto.ParticipantResponse{
			UserID: p.UserID,
			Role:   p.Role,
		})
	}
	return resp
}

func parseUUIDs(strs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(strs))
	for i, s := range strs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}
