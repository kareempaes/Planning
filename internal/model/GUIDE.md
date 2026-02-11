# model/ — Domain Types

This package defines the core data structures shared across all layers (repo, service, handler). It has **no dependencies on other internal packages** — only stdlib and `google/uuid`. This is intentional: it sits at the bottom of the import graph so every layer can use it without circular imports.

## Files

### `user.go` — COMPLETE

| Type | Purpose |
|------|---------|
| `User` | Full database row representation. `PasswordHash` is tagged `json:"-"` so it's never accidentally serialized to an API response. |
| `PublicProfile` | The subset returned by `GET /users/:id` — strips email, password hash, and timestamps. Exists because the API deliberately hides private fields from other users. |
| `UpdateProfileParams` | Uses `*string` pointer fields so the caller can distinguish "not sent" (nil) from "set to empty" (""). This matters for PATCH semantics where omitted fields should remain unchanged. |
| `UserSearchResult` | Lightweight struct for search results — no email, no status. Keeps search payloads small. |
| `Page[T]` | Generic paginated wrapper using Go generics. Reused across all paginated endpoints (users, conversations, messages). `NextCursor` is an opaque string the client passes back to get the next page. |

### `session.go` — COMPLETE

| Type | Purpose |
|------|---------|
| `Session` | Represents a refresh token entry. `RefreshTokenHash` stores a hash (never the raw token) for security. `RevokedAt` is `*time.Time` — nil means the session is still active. |

**Why sessions are separate from users:** A user can have multiple active sessions (different devices). The session table tracks each refresh token independently, enabling selective revocation (logout one device) or bulk revocation (logout everywhere).

### `conversation.go` — COMPLETE

| Type | Purpose |
|------|---------|
| `Conversation` | Core conversation record. `Type` is `"direct"` or `"group"`. `Name` is `*string` because direct conversations don't have names (nil), while group conversations do. |
| `ConversationParticipant` | Join table record linking users to conversations. `Role` is `"owner"` or `"member"` — only group creators are owners. `LeftAt` is nil while the user is still in the conversation. |
| `ConversationSummary` | The shape returned by `GET /conversations` (list view). Includes the last message preview and unread count so the client can render an inbox without fetching full message histories. |
| `ParticipantSummary` | Minimal participant info for the conversation list — just user ID and display name. |
| `MessagePreview` | Truncated message for the conversation list — just enough to show "Alice: Hey there..." in the inbox. |

### `message.go` — COMPLETE

| Type | Purpose |
|------|---------|
| `Message` | A chat message. `Status` tracks the overall message state (`"sent"`, `"delivered"`). `ConversationID` scopes it to a conversation; `SenderID` identifies who sent it. |
| `MessageDelivery` | Per-recipient delivery tracking. In a group of 5 people, one message creates 4 delivery rows (one per other participant). This enables "delivered to 3 of 4" granularity and future read receipts. |

**Why separate Message and MessageDelivery:** A message is written once but delivered to N recipients. Tracking delivery per-user requires a separate table. The `status` on `Message` is a denormalized summary; the authoritative state lives in `MessageDelivery`.

### `moderation.go` — COMPLETE

| Type | Purpose |
|------|---------|
| `BlockedUser` | Records that one user blocked another. Blocking is directional — A blocks B doesn't mean B blocks A. |
| `Report` | A user-submitted report. `TargetType` is `"user"`, `"message"`, or `"conversation"` with `TargetID` being the UUID of the target. This polymorphic approach avoids separate report tables per entity. |

### `errors.go` — COMPLETE

| Type | Purpose |
|------|---------|
| `ErrNotFound` | Sentinel error for missing resources. The handler layer maps this to HTTP 404. |
| `ErrValidation` | Sentinel for invalid input. Maps to HTTP 422. |
| `ErrConflict` | Sentinel for uniqueness violations (e.g., duplicate email). Maps to HTTP 409. |
| `ValidationError` | Carries field-level detail (`Field` + `Message`). Wraps `ErrValidation` via `Unwrap()` so callers can use both `errors.Is(err, model.ErrValidation)` and `errors.As(err, &ve)` to extract the field name. |

**Why sentinel errors in the model package:** Domain errors are defined here (not in repo or service) so every layer shares the same error vocabulary. The repo returns `ErrNotFound`, the service passes it through, and the handler maps it to a status code — no layer-specific error types needed.
