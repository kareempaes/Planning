package service

import (
	"fmt"

	"github.com/kareempaes/planning/internal/repo"
)

// RegistryType identifies a service configuration variant.
type RegistryType int

const (
	DefaultRegistry RegistryType = iota
)

// Registry aggregates all service instances.
type Registry struct {
	Users         *UserService
	Auth          *AuthService
	Conversations *ConversationService
	Messages      *MessageService
	Moderation    *ModerationService
}

// NewRegistry creates a Registry based on the given configuration type.
func NewRegistry(regType RegistryType, store *repo.Store, authCfg AuthConfig) (*Registry, error) {
	switch regType {
	case DefaultRegistry:
		return &Registry{
			Users:         NewUserService(store.Users),
			Auth:          NewAuthService(store.Users, store.Sessions, authCfg),
			Conversations: NewConversationService(store.Conversations),
			Messages:      NewMessageService(store.Messages, store.Conversations),
			Moderation:    NewModerationService(store.Moderation),
		}, nil
	default:
		return nil, fmt.Errorf("unknown registry type: %d", regType)
	}
}
