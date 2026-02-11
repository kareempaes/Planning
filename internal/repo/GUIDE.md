# repo/ — Repository Layer

This package owns all SQL queries. It translates between the database and domain models. Each file defines an **interface** (the contract) and a **concrete struct** (the SQL implementation). The service layer depends only on the interface, never on the concrete type.

## Architecture Pattern

```
Service calls repo interface method
  → concrete struct executes SQL via *sql.DB
    → scans result into model struct
      → returns model (or domain error)
```

**Why interfaces:** Testability. In unit tests, you can mock `UserRepository` without touching a database. In integration tests, you use the real SQL implementation against an in-memory SQLite.

## Files

### `user.go` — COMPLETE (reference implementation)

Study this file as the pattern to follow for all other repos.

| Method | SQL | Notes |
|--------|-----|-------|
| `Create` | `INSERT INTO users` | Detects unique constraint violations and returns `model.ErrConflict` |
| `GetByID` | `SELECT ... WHERE id = $1` | Returns `model.ErrNotFound` on `sql.ErrNoRows` |
| `GetByEmail` | `SELECT ... WHERE email = $1` | Same error mapping |
| `Update` | Dynamic `UPDATE ... SET ... RETURNING` | Builds SET clause from non-nil fields in `UpdateProfileParams`. Always bumps `updated_at`. |
| `Search` | `SELECT ... WHERE display_name ILIKE $1 ORDER BY display_name, id LIMIT $N` | Cursor-based pagination using `(display_name, id)` keyset. Fetches `limit+1` to determine `has_more`. |

**Key patterns in user.go:**
- **Error mapping:** `sql.ErrNoRows` → `model.ErrNotFound`. String-check for "unique"/"duplicate" → `model.ErrConflict`. All other errors get wrapped with `fmt.Errorf("repo: context: %w", err)`.
- **Dynamic SQL:** The `Update` method builds parameterized SQL at runtime. Only non-nil fields become SET clauses. Parameter indices (`$1`, `$2`, ...) are tracked with a counter.
- **Cursor encoding:** `encodeCursor` / `decodeCursor` use base64-encoded `"displayName|uuid"` strings. The cursor is opaque to the client.

### `session.go` — STUB (interface + panic stubs)

| Method | What it should do |
|--------|-------------------|
| `Create` | `INSERT INTO sessions` — store the hashed refresh token, user ID, and expiry. |
| `GetByToken` | `SELECT ... WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > now()` — find an active session by token hash. |
| `Revoke` | `UPDATE sessions SET revoked_at = now() WHERE id = $1` — soft-revoke a single session. |
| `RevokeAllForUser` | `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL` — logout everywhere. |

**Why hash the refresh token:** Refresh tokens are long-lived secrets. If the database is compromised, storing hashes (not raw tokens) prevents an attacker from using stolen tokens. Hash with SHA-256 before storing/querying.

### `conversation.go` — STUB (interface + panic stubs)

| Method | What it should do |
|--------|-------------------|
| `Create` | `INSERT INTO conversations` — create the conversation row. |
| `GetByID` | `SELECT` the conversation. Consider joining participants. |
| `ListByUser` | Complex query: JOIN `conversation_participants` to find the user's conversations, LEFT JOIN latest message, COUNT unread deliveries. Cursor-paginated. |
| `Update` | `UPDATE conversations SET name = $1` — only for group renaming. |
| `FindDirectBetween` | Find an existing direct conversation between two specific users. Used during `POST /conversations` to return 200 instead of creating a duplicate. |
| `AddParticipant` | `INSERT INTO conversation_participants`. |
| `RemoveParticipant` | `UPDATE conversation_participants SET left_at = now()` — soft removal, not a hard delete, so history is preserved. |
| `GetParticipants` | `SELECT` all active participants (where `left_at IS NULL`). |
| `IsParticipant` | Quick existence check — used to authorize actions (can this user send a message here?). |

**Why `FindDirectBetween`:** The API spec says creating a direct conversation that already exists returns the existing one (200) instead of creating a duplicate (201). This method enables that idempotency check.

### `message.go` — STUB (interface + panic stubs)

| Method | What it should do |
|--------|-------------------|
| `Create` | `INSERT INTO messages` — persist the message. |
| `GetByID` | `SELECT` a single message by ID. |
| `ListByConversation` | `SELECT ... WHERE conversation_id = $1 ORDER BY created_at DESC` — newest first, cursor-paginated. The cursor here should be `(created_at, id)` since messages are ordered by time. |
| `CreateDeliveries` | Batch `INSERT INTO message_deliveries` — one row per recipient (excluding the sender). |
| `UpdateDeliveryStatus` | `UPDATE message_deliveries SET status = $1, delivered_at = now()` — called when the WebSocket receives an `ack` frame from a client. |

**Why separate CreateDeliveries:** When a message is sent in a group of 5, you need 4 delivery tracking rows. Batch-inserting them in one call is more efficient than 4 separate inserts.

### `moderation.go` — STUB (interface + panic stubs)

| Method | What it should do |
|--------|-------------------|
| `Block` | `INSERT INTO blocked_users` — create the block. Should be idempotent (no error if already blocked). |
| `Unblock` | `DELETE FROM blocked_users WHERE blocker_id = $1 AND blocked_id = $2`. |
| `ListBlocked` | `SELECT` all users blocked by a given user. Join with `users` table to get display names. |
| `IsBlocked` | Quick existence check — used by the message service to prevent sending messages to someone who blocked you. |
| `CreateReport` | `INSERT INTO reports`. |

### `state.go` — COMPLETE

The `Store` struct aggregates all repository instances. The `NewStore` factory initializes them all from a single `*sql.DB`. This is the repo layer's entry point — `main.go` calls `NewStore` and passes the result to the service layer.

```
Store.Users         → NewUserRepo(db)
Store.Sessions      → NewSessionRepo(db)
Store.Conversations → NewConversationRepo(db)
Store.Messages      → NewMessageRepo(db)
Store.Moderation    → NewModerationRepo(db)
```

## Common Patterns to Follow

1. **Always use `context.Context`** as the first parameter — enables cancellation and timeouts
2. **Map `sql.ErrNoRows` to `model.ErrNotFound`** — don't leak database errors to the service layer
3. **Use `$1, $2, ...` placeholders** — works with both pgx and modernc/sqlite
4. **Wrap errors with context** — `fmt.Errorf("repo: get user by id: %w", err)` makes debugging easier
5. **Use `RETURNING` for updates** — avoids a separate SELECT after UPDATE
6. **Cursor pagination over offset** — offsets get slower as page depth increases; keyset cursors are O(1)
