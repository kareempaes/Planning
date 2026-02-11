# service/ — Service Layer

This package contains business logic. Each service wraps one or more repository interfaces and adds validation, authorization checks, and data transformation. Services know nothing about HTTP, SQL, or WebSockets — they work with domain models and domain errors.

## Architecture Pattern

```
Handler parses HTTP request
  → calls service method with domain types
    → service validates input
    → service calls repo method(s)
    → service transforms/filters the result
  → handler marshals response to JSON
```

**Why a separate service layer:** Keeps business rules in one place. If you add a CLI, a gRPC API, or a background job, they all call the same service methods. The rules (e.g., "display name must be <= 100 chars") are defined once, not duplicated across handlers.

## Files

### `user.go` — COMPLETE (reference implementation)

Study this file as the pattern to follow for all other services.

| Method | Business Logic |
|--------|----------------|
| `GetProfile` | Thin passthrough — just calls `repo.GetByID`. Returns the full user including email. |
| `UpdateProfile` | Trims whitespace, validates display_name (non-empty, <= 100 chars), validates avatar_url (<= 2048 chars), ensures at least one field is provided. Then delegates to repo. |
| `GetPublicProfile` | Calls `repo.GetByID`, then maps to `model.PublicProfile` — stripping email, password hash, and timestamps. This is the service's job, not the repo's. |
| `SearchUsers` | Validates the search query (non-empty, <= 100 chars), clamps limit to [1, 100] with default 20. Delegates to repo. |

**Key pattern:** The service validates and transforms, the repo executes SQL. If validation fails, the service returns a `model.ValidationError` — the handler maps that to HTTP 422.

### `auth.go` — STUB (constructor only)

| Method to implement | Business Logic |
|---------------------|----------------|
| `Register` | Validate email format, check password strength, hash password with bcrypt, call `users.Create`, create a session, generate JWT access token + refresh token, return user + tokens. |
| `Login` | Call `users.GetByEmail`, compare password with bcrypt, create session, generate tokens. |
| `RefreshToken` | Hash the incoming refresh token, call `sessions.GetByToken`, verify not expired/revoked, generate new token pair, revoke old session, create new session. |
| `Logout` | Hash the refresh token, find the session, call `sessions.Revoke`. |
| `ForgotPassword` | Look up user by email, generate a reset token, send email. Always return 202 (don't leak whether the email exists). |
| `ResetPassword` | Validate the reset token, hash the new password, update the user. |

**Dependencies:** This service needs both `UserRepository` and `SessionRepository` — that's why the constructor takes both. It will also need a JWT signing key (passed via config) and a bcrypt cost factor.

**Why auth is a service, not middleware:** Authentication (verifying credentials, issuing tokens) is business logic. Authorization (checking if a request has a valid token) is middleware. The auth *service* handles login/register/refresh. JWT *middleware* in the handler layer validates tokens on every request.

### `conversation.go` — STUB (constructor only)

| Method to implement | Business Logic |
|---------------------|----------------|
| `Create` | Validate participant IDs exist. For `type: "direct"`, call `FindDirectBetween` first — return existing if found. Otherwise create conversation + add all participants in a transaction. |
| `List` | Call `repo.ListByUser` with cursor pagination. |
| `GetByID` | Verify the caller is a participant (authorization), then return conversation details. |
| `Update` | Verify the caller is the owner (for group rename). Call `repo.Update`. |
| `AddParticipants` | Verify the caller is a participant. Add new users. |
| `RemoveParticipant` | Verify the caller is the owner or is removing themselves. Call `repo.RemoveParticipant`. |

**Why authorization lives here:** The service checks "is this user allowed to do this?" by calling `repo.IsParticipant`. This isn't input validation (that's format checking) — it's a business rule about who can access what.

### `message.go` — STUB (constructor only)

| Method to implement | Business Logic |
|---------------------|----------------|
| `Send` | Verify sender is a participant. Create the message. Create delivery rows for all other participants. Publish an event to the WebSocket hub for real-time delivery. |
| `GetHistory` | Verify the caller is a participant. Call `repo.ListByConversation` with cursor pagination. |
| `GetByID` | Verify the caller is a participant. Return the single message. |

**Dependencies:** Takes both `MessageRepository` and `ConversationRepository` — needs the conversation repo to check participant membership before allowing message operations.

**Why the message service checks participation:** Without this check, any authenticated user could read any conversation's messages by guessing the conversation ID. The participant check is the authorization boundary.

### `moderation.go` — STUB (constructor only)

| Method to implement | Business Logic |
|---------------------|----------------|
| `Block` | Validate that the user isn't blocking themselves. Call `repo.Block`. Optionally: auto-leave shared conversations. |
| `Unblock` | Call `repo.Unblock`. |
| `ListBlocked` | Call `repo.ListBlocked`. |
| `Report` | Validate the target exists (user/message/conversation). Create the report. |

**Why moderation is its own service:** Block and report logic crosses domains (affects conversations, messages, user visibility). Keeping it separate avoids circular dependencies between conversation and message services.

### `state.go` — COMPLETE

The `Registry` struct aggregates all service instances. The `NewRegistry` factory wires each service to its repository dependencies from the `repo.Store`:

```
Registry.Users         → NewUserService(store.Users)
Registry.Auth          → NewAuthService(store.Users, store.Sessions)
Registry.Conversations → NewConversationService(store.Conversations)
Registry.Messages      → NewMessageService(store.Messages, store.Conversations)
Registry.Moderation    → NewModerationService(store.Moderation)
```

Note how `AuthService` gets the Users repo (to create/look up users) and the Sessions repo (to manage tokens). `MessageService` gets both Messages and Conversations repos (to verify participant membership).

## Common Patterns to Follow

1. **Validate first, then delegate** — check inputs before calling the repo
2. **Return domain errors** — `model.ValidationError` for bad input, let `model.ErrNotFound` from the repo pass through
3. **Don't catch repo errors you can't handle** — wrapping them adds noise. Just let them propagate.
4. **Authorization = business logic** — "is this user a participant?" is a service concern, not middleware
5. **Keep services thin when possible** — if there's no validation or transformation to add, a simple passthrough to the repo is fine (see `GetProfile`)
