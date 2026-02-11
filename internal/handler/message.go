package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kareempaes/planning/internal/dto"
	"github.com/kareempaes/planning/internal/infra"
	"github.com/kareempaes/planning/internal/model"
	"github.com/kareempaes/planning/internal/service"
)

// MessageHandler handles message endpoints.
type MessageHandler struct {
	messages *service.MessageService
	convos   *service.ConversationService
	hub      *infra.Hub
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(messages *service.MessageService, convos *service.ConversationService, hub *infra.Hub) *MessageHandler {
	return &MessageHandler{messages: messages, convos: convos, hub: hub}
}

// Send handles POST /conversations/{id}/messages.
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid conversation ID"},
		})
		return
	}

	var req dto.SendMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid request body"},
		})
		return
	}

	msg, err := h.messages.Send(r.Context(), userID, convoID, req.Body)
	if err != nil {
		writeError(w, err)
		return
	}

	// Push real-time event to other participants.
	go h.broadcastMessage(r.Context(), userID, convoID, msg)

	writeJSON(w, http.StatusCreated, toMessageResponse(msg))
}

// GetHistory handles GET /conversations/{id}/messages.
func (h *MessageHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid conversation ID"},
		})
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	page, err := h.messages.GetHistory(r.Context(), userID, convoID, cursor, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	msgs := make([]dto.MessageResponse, len(page.Items))
	for i, m := range page.Items {
		msgs[i] = toMessageResponse(&m)
	}

	writeJSON(w, http.StatusOK, dto.MessageListResponse{
		Messages: msgs,
		Pagination: dto.PaginationResponse{
			NextCursor: page.NextCursor,
			HasMore:    page.HasMore,
		},
	})
}

// GetByID handles GET /conversations/{id}/messages/{messageId}.
func (h *MessageHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	convoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid conversation ID"},
		})
		return
	}

	msgID, err := uuid.Parse(chi.URLParam(r, "messageId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{
			Error: ErrorDetail{Code: "bad_request", Message: "invalid message ID"},
		})
		return
	}

	msg, err := h.messages.GetByID(r.Context(), userID, convoID, msgID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMessageResponse(msg))
}

func (h *MessageHandler) broadcastMessage(ctx context.Context, senderID uuid.UUID, convoID uuid.UUID, msg *model.Message) {
	_, participants, err := h.convos.GetByID(ctx, senderID, convoID)
	if err != nil {
		return
	}

	recipientIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		if p.UserID != senderID {
			recipientIDs = append(recipientIDs, p.UserID)
		}
	}

	if len(recipientIDs) == 0 {
		return
	}

	data, _ := json.Marshal(toMessageResponse(msg))
	h.hub.SendToUsers(recipientIDs, infra.Event{
		Type: "message",
		Data: data,
	})
}

func toMessageResponse(m *model.Message) dto.MessageResponse {
	return dto.MessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		Status:         m.Status,
		CreatedAt:      m.CreatedAt,
	}
}
