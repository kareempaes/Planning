package handler

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kareempaes/planning/internal/infra"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // TODO: restrict in production
}

// WSHandler handles WebSocket upgrade requests.
type WSHandler struct {
	hub       *infra.Hub
	jwtSecret string
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *infra.Hub, jwtSecret string) *WSHandler {
	return &WSHandler{hub: hub, jwtSecret: jwtSecret}
}

// Upgrade handles GET /ws — upgrades to a WebSocket connection.
func (h *WSHandler) Upgrade(w http.ResponseWriter, r *http.Request) {
	// Accept token from query param or Authorization header.
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// Fallback: the AuthMiddleware already validated the header.
		// Extract user ID from context if present.
		if uid, ok := r.Context().Value(userIDKey).(uuid.UUID); ok {
			h.upgradeConnection(w, r, uid)
			return
		}
		writeJSON(w, http.StatusUnauthorized, ErrorBody{
			Error: ErrorDetail{Code: "unauthorized", Message: "missing token"},
		})
		return
	}

	// Validate JWT from query param.
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		writeJSON(w, http.StatusUnauthorized, ErrorBody{
			Error: ErrorDetail{Code: "unauthorized", Message: "invalid or expired token"},
		})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, ErrorBody{
			Error: ErrorDetail{Code: "unauthorized", Message: "invalid token claims"},
		})
		return
	}

	sub, err := claims.GetSubject()
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorBody{
			Error: ErrorDetail{Code: "unauthorized", Message: "missing subject claim"},
		})
		return
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorBody{
			Error: ErrorDetail{Code: "unauthorized", Message: "invalid user ID in token"},
		})
		return
	}

	h.upgradeConnection(w, r, userID)
}

func (h *WSHandler) upgradeConnection(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &infra.Client{
		Hub:    h.hub,
		Conn:   conn,
		UserID: userID,
		Send:   make(chan []byte, 256),
	}

	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
