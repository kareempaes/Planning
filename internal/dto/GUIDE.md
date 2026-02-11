# dto/ — Data Transfer Objects

This package holds the structs that sit between external boundaries and the domain model. There are two categories:

## DB DTOs (`db_*.go`)

These are row-scanning structs that match the exact shape of SQL query results. They decouple your SQL layer from your domain model.

**Why not scan directly into `model.User`?**

You can — and `repo/user.go` currently does. But as queries get more complex (JOINs, aggregations, computed columns), the result shape stops matching the domain model. A DB DTO lets you:

1. Scan a JOIN result into a flat struct without polluting the domain model with nullable join fields
2. Handle SQL-specific types (e.g., `sql.NullString`) in one place, then convert to clean Go types
3. Keep the domain model stable even when queries change

### Files

| File | Maps to table(s) | Notes |
|------|-------------------|-------|
| `db_user.go` | `users` | Row struct for user queries. Consider adding a `UserRow` struct with `sql.NullString` for `avatar_url`. |
| `db_session.go` | `sessions` | Row struct for session queries. `revoked_at` is nullable — use `sql.NullTime`. |
| `db_conversation.go` | `conversations`, `conversation_participants` | May need multiple structs: one for the conversation row, one for the joined participant row, one for the summary query (which JOINs messages + participants). |
| `db_message.go` | `messages`, `message_deliveries` | Separate structs for message rows vs delivery rows. The "list messages" query may join deliveries. |
| `db_moderation.go` | `blocked_users`, `reports` | Row structs for block and report queries. |

### Pattern

```go
// db_user.go
type UserRow struct {
    ID           uuid.UUID
    Email        string
    PasswordHash string
    DisplayName  string
    AvatarURL    sql.NullString  // handles NULL from DB
    Status       string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// Convert to domain model
func (r *UserRow) ToModel() *model.User {
    var avatar *string
    if r.AvatarURL.Valid {
        avatar = &r.AvatarURL.String
    }
    return &model.User{
        ID:           r.ID,
        AvatarURL:    avatar,
        // ... etc
    }
}
```

## API DTOs (`api_*.go`)

These are the JSON request/response structs that match the API contract in `docs/API.md`. They decouple your HTTP layer from the domain model.

**Why not marshal `model.User` directly?**

The API contract and the domain model will diverge:

1. **Different field names:** The API uses `snake_case` JSON, but you may want different Go field names internally
2. **Different shapes:** `POST /auth/register` returns `{ "user": {...}, "tokens": {...} }` — that's not a `model.User`, it's a composite response
3. **Input validation tags:** API DTOs can carry `validate:"required"` struct tags without cluttering the domain model
4. **Versioning:** If you ever need `/api/v2` with a different response shape, the domain model stays unchanged

### Files

| File | Covers endpoints | Notes |
|------|------------------|-------|
| `api_auth.go` | `POST /auth/register`, `/login`, `/refresh`, `/logout`, `/forgot-password`, `/reset-password` | Request structs (email+password, refresh token) and response structs (user+tokens). |
| `api_user.go` | `GET /users/me`, `PATCH /users/me`, `GET /users/:id`, `GET /users?q=` | Request struct for PATCH (partial update), response structs for full profile vs public profile vs search results. |
| `api_conversation.go` | `POST /conversations`, `GET /conversations`, `GET /conversations/:id`, `PATCH /conversations/:id`, participant endpoints | Request structs for create/update, response structs for conversation detail vs list summary. |
| `api_message.go` | `POST /conversations/:id/messages`, `GET .../messages`, `GET .../messages/:id` | Request struct for send (just `body`), response structs for message and paginated list. |
| `api_moderation.go` | `POST /users/:id/block`, `DELETE .../block`, `GET /users/me/blocked`, `POST /reports` | Request struct for report (target_type, target_id, reason), response structs for blocked list and report confirmation. |

### Pattern

```go
// api_auth.go
type RegisterRequest struct {
    Email       string `json:"email"`
    Password    string `json:"password"`
    DisplayName string `json:"display_name"`
}

type RegisterResponse struct {
    User   UserResponse  `json:"user"`
    Tokens TokenResponse `json:"tokens"`
}

type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
}
```

## When to Use DTOs vs Direct Model Mapping

- **Simple CRUD with no joins:** Scanning directly into `model.User` is fine (like `repo/user.go` does now)
- **Complex queries or API responses:** Use DTOs to keep boundaries clean
- **Rule of thumb:** If you find yourself adding `json` tags or `sql.Null*` types to a model struct just for one use case, that's a sign you need a DTO
