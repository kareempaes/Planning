package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kareempaes/planning/internal/infra"
	"github.com/kareempaes/planning/internal/service"
)

// NewRouter creates the chi router with all API routes.
func NewRouter(registry *service.Registry, hub *infra.Hub, jwtSecret string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/api/v1", func(r chi.Router) {
		auth := NewAuthHandler(registry.Auth)
		r.Post("/auth/register", auth.Register)
		r.Post("/auth/login", auth.Login)
		r.Post("/auth/refresh", auth.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(jwtSecret))

			r.Post("/auth/logout", auth.Logout)

			users := NewUserHandler(registry.Users)
			mod := NewModerationHandler(registry.Moderation, registry.Users)

			r.Get("/users/me", users.GetMe)
			r.Patch("/users/me", users.UpdateMe)
			r.Get("/users/me/blocked", mod.ListBlocked)
			r.Get("/users/{id}", users.GetPublicProfile)
			r.Get("/users", users.Search)
			r.Post("/users/{id}/block", mod.Block)
			r.Delete("/users/{id}/block", mod.Unblock)

			convos := NewConversationHandler(registry.Conversations)
			r.Post("/conversations", convos.Create)
			r.Get("/conversations", convos.List)
			r.Get("/conversations/{id}", convos.GetByID)
			r.Patch("/conversations/{id}", convos.Update)
			r.Post("/conversations/{id}/participants", convos.AddParticipants)
			r.Delete("/conversations/{id}/participants/{userId}", convos.RemoveParticipant)

			msgs := NewMessageHandler(registry.Messages, registry.Conversations, hub)
			r.Post("/conversations/{id}/messages", msgs.Send)
			r.Get("/conversations/{id}/messages", msgs.GetHistory)
			r.Get("/conversations/{id}/messages/{messageId}", msgs.GetByID)

			r.Post("/reports", mod.Report)

			ws := NewWSHandler(hub, jwtSecret)
			r.Get("/ws", ws.Upgrade)
		})
	})

	return r
}
