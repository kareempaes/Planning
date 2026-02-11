package repo

import (
	"database/sql"
	"fmt"
)

// StoreType identifies a supported repository backend.
type StoreType int

const (
	SQLStore StoreType = iota
)

// Store aggregates all repository instances.
type Store struct {
	Users         UserRepository
	Sessions      SessionRepository
	Conversations ConversationRepository
	Messages      MessageRepository
	Moderation    ModerationRepository
}

// NewStore creates a Store based on the given backend type.
func NewStore(storeType StoreType, db *sql.DB) (*Store, error) {
	switch storeType {
	case SQLStore:
		return &Store{
			Users:         NewUserRepo(db),
			Sessions:      NewSessionRepo(db),
			Conversations: NewConversationRepo(db),
			Messages:      NewMessageRepo(db),
			Moderation:    NewModerationRepo(db),
		}, nil
	default:
		return nil, fmt.Errorf("unknown store type: %d", storeType)
	}
}
