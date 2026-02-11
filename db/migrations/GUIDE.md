# db/migrations/ — Database Migrations

This directory holds SQL migration files in [golang-migrate](https://github.com/golang-migrate/migrate) format. Migrations are applied automatically at application startup via `infra.RunMigrations()`.

## File Naming Convention

```
NNNNNN_description.up.sql    — applies the migration (forward)
NNNNNN_description.down.sql  — reverts the migration (rollback)
```

The `NNNNNN` is a zero-padded sequence number. golang-migrate applies them in order and tracks which have been applied in a `schema_migrations` table it creates automatically.

## Current State

### `000001_init_schema` — COMPLETE

Creates the foundational tables:

**`users`** — User accounts
- `id UUID PK` with `gen_random_uuid()` default (requires `pgcrypto` extension)
- `email` UNIQUE — for login
- `password_hash` — bcrypt hash, never the raw password
- `display_name` — shown in conversations
- `avatar_url` — nullable profile picture URL
- `status` — defaults to `'offline'`, updated by the WebSocket layer
- `created_at`, `updated_at` — audit timestamps
- Indexes: `email` (login lookup), `display_name` (search), `(display_name, id)` (cursor pagination)

**`sessions`** — Refresh token storage
- `user_id FK → users(id) ON DELETE CASCADE` — deleting a user deletes all their sessions
- `refresh_token_hash` UNIQUE — stores a SHA-256 hash, not the raw token
- `expires_at` — refresh tokens have a finite lifetime
- `revoked_at` — nullable, set when the user logs out (soft revocation)
- Index: `user_id` (find all sessions for a user)

## Migrations Still Needed

### `000002_conversations` — TODO

```sql
CREATE TABLE conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       VARCHAR(20) NOT NULL,          -- 'direct' or 'group'
    name       VARCHAR(100),                   -- NULL for direct, required for group
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE conversation_participants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL DEFAULT 'member',  -- 'owner' or 'member'
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at         TIMESTAMPTZ,                            -- NULL = still active

    CONSTRAINT unique_active_participant UNIQUE (conversation_id, user_id)
);

CREATE INDEX idx_cp_conversation_id ON conversation_participants (conversation_id);
CREATE INDEX idx_cp_user_id ON conversation_participants (user_id);
```

**Why `left_at` instead of DELETE:** Preserving the participant record lets you show "Alice left the group" in chat history and prevents re-joining issues.

**Why UNIQUE on (conversation_id, user_id):** Prevents duplicate memberships. If a user leaves and rejoins, you'd update `left_at` back to NULL rather than inserting a new row.

### `000003_messages` — TODO

```sql
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'sent',  -- 'sent', 'delivered'
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation_id ON messages (conversation_id, created_at DESC);

CREATE TABLE message_deliveries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id   UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',  -- 'pending', 'delivered', 'read'
    delivered_at TIMESTAMPTZ,
    read_at      TIMESTAMPTZ,

    CONSTRAINT unique_delivery UNIQUE (message_id, user_id)
);

CREATE INDEX idx_md_message_id ON message_deliveries (message_id);
CREATE INDEX idx_md_user_id ON message_deliveries (user_id);
```

**Why the compound index `(conversation_id, created_at DESC)`:** The most common query is "get recent messages in this conversation" — this index covers it without a sort step.

**Why `message_deliveries` is separate from `messages`:** One message in a group of N people needs N-1 delivery tracking rows. Embedding delivery status in the message row doesn't work for groups.

### `000004_moderation` — TODO

```sql
CREATE TABLE blocked_users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT unique_block UNIQUE (blocker_id, blocked_id),
    CONSTRAINT no_self_block CHECK (blocker_id != blocked_id)
);

CREATE INDEX idx_blocked_blocker ON blocked_users (blocker_id);

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id),
    target_type VARCHAR(20) NOT NULL,  -- 'user', 'message', 'conversation'
    target_id   UUID NOT NULL,         -- polymorphic FK (not enforced by DB)
    reason      TEXT NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_reports_reporter ON reports (reporter_id);
CREATE INDEX idx_reports_target ON reports (target_type, target_id);
```

**Why `CHECK (blocker_id != blocked_id)`:** Prevents a user from blocking themselves — a data integrity guard that complements the service-layer validation.

**Why `target_id` is not a FK:** Reports can target users, messages, or conversations. A polymorphic foreign key can't reference multiple tables. The service layer validates that the target exists before creating the report.

## SQLite Compatibility Notes

The migration files use PostgreSQL-specific syntax (`gen_random_uuid()`, `TIMESTAMPTZ`, `pgcrypto`). For SQLite dev/test:
- UUIDs must be generated in Go code (`uuid.New()`) and passed explicitly — SQLite has no `gen_random_uuid()`
- `TIMESTAMPTZ` is stored as TEXT in SQLite — the Go driver handles parsing
- Consider maintaining a separate SQLite-compatible schema for tests, or apply schema directly in test setup code
