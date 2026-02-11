---
title: Chat Application API
tags:
  - api
  - rest
  - websocket
status: draft
---

# Chat Application API

**Base URL:** `/api/v1`

**Authentication:** Bearer JWT in the `Authorization` header. Routes marked "public" do not require a token.

**Common response conventions:**
- Timestamps: ISO 8601 / RFC 3339 with timezone
- Pagination: cursor-based (`?cursor=<opaque>&limit=N`)
- Error shape: `{ "error": { "code": "string", "message": "string" } }`

---

## Route Map

![[docs/diagrams/api-routes.md]]

---

## Authentication

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | Public | Create a new account |
| POST | `/auth/login` | Public | Log in with email/password |
| POST | `/auth/refresh` | Public | Refresh an access token |
| POST | `/auth/logout` | Yes | Revoke a refresh token |
| POST | `/auth/forgot-password` | Public | Request a password-reset email |
| POST | `/auth/reset-password` | Public | Reset password with token |

### POST `/auth/register`

```jsonc
// Request
{ "email": "string", "password": "string", "display_name": "string" }

// 201 Response
{
  "user": { "id": "uuid", "email": "string", "display_name": "string", "created_at": "iso8601" },
  "tokens": { "access_token": "string", "refresh_token": "string", "expires_in": 900 }
}
```

### POST `/auth/login`

```jsonc
// Request
{ "email": "string", "password": "string" }

// 200 Response — same shape as /auth/register
```

### POST `/auth/refresh`

```jsonc
// Request
{ "refresh_token": "string" }

// 200 Response
{ "access_token": "string", "refresh_token": "string", "expires_in": 900 }
```

### POST `/auth/logout`

```jsonc
// Request
{ "refresh_token": "string" }

// 204 No Content
```

### POST `/auth/forgot-password`

```jsonc
// Request
{ "email": "string" }

// 202 Accepted (always, to avoid email enumeration)
```

### POST `/auth/reset-password`

```jsonc
// Request
{ "token": "string", "new_password": "string" }

// 200 Response
{ "message": "Password reset successfully" }
```

---

## Users

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/users/me` | Yes | Get current user's profile |
| PATCH | `/users/me` | Yes | Update current user's profile |
| GET | `/users/:id` | Yes | Get a user's public profile |
| GET | `/users?q=` | Yes | Search users by name or email |

### GET `/users/me`

```jsonc
// 200 Response
{ "id": "uuid", "email": "string", "display_name": "string", "avatar_url": "string|null", "status": "online|offline", "created_at": "iso8601" }
```

### PATCH `/users/me`

```jsonc
// Request (all fields optional)
{ "display_name": "string", "avatar_url": "string" }

// 200 Response — updated user object
```

### GET `/users/:id`

```jsonc
// 200 Response (public fields only)
{ "id": "uuid", "display_name": "string", "avatar_url": "string|null", "status": "online|offline" }
```

### GET `/users?q=search_term&cursor=&limit=20`

```jsonc
// 200 Response
{
  "users": [{ "id": "uuid", "display_name": "string", "avatar_url": "string|null" }],
  "pagination": { "next_cursor": "string|null", "has_more": true }
}
```

---

## Conversations

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/conversations` | Yes | Create a conversation |
| GET | `/conversations` | Yes | List current user's conversations |
| GET | `/conversations/:id` | Yes | Get conversation details |
| PATCH | `/conversations/:id` | Yes | Update conversation (rename group) |
| POST | `/conversations/:id/participants` | Yes | Add participants to a group |
| DELETE | `/conversations/:id/participants/:userId` | Yes | Remove a participant from a group |

### POST `/conversations`

```jsonc
// Request
{ "type": "direct|group", "participant_ids": ["uuid"], "name": "string (optional, group only)" }

// 201 Response
{
  "id": "uuid",
  "type": "direct|group",
  "name": "string|null",
  "participants": [{ "user_id": "uuid", "display_name": "string", "role": "owner|member" }],
  "created_at": "iso8601"
}
```

For `type: "direct"`, if a 1:1 conversation already exists between the two users, returns the existing conversation with `200` instead of `201`.

### GET `/conversations?cursor=&limit=20`

```jsonc
// 200 Response
{
  "conversations": [{
    "id": "uuid",
    "type": "direct|group",
    "name": "string|null",
    "last_message": { "id": "uuid", "body": "string", "sender_id": "uuid", "created_at": "iso8601" },
    "unread_count": 3,
    "participants": [{ "user_id": "uuid", "display_name": "string" }]
  }],
  "pagination": { "next_cursor": "string|null", "has_more": true }
}
```

### GET `/conversations/:id`

```jsonc
// 200 Response — full conversation object with participants list
```

### PATCH `/conversations/:id`

```jsonc
// Request
{ "name": "string" }

// 200 Response — updated conversation object
```

### POST `/conversations/:id/participants`

```jsonc
// Request
{ "user_ids": ["uuid"] }

// 200 Response — updated participants list
```

### DELETE `/conversations/:id/participants/:userId`

```jsonc
// 204 No Content
```

---

## Messages

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/conversations/:id/messages` | Yes | Send a message |
| GET | `/conversations/:id/messages` | Yes | Get message history (paginated) |
| GET | `/conversations/:id/messages/:messageId` | Yes | Get a single message |

### POST `/conversations/:id/messages`

```jsonc
// Request
{ "body": "string" }

// 201 Response
{
  "id": "uuid",
  "conversation_id": "uuid",
  "sender_id": "uuid",
  "body": "string",
  "status": "sent",
  "created_at": "iso8601"
}
```

### GET `/conversations/:id/messages?cursor=&limit=50`

Returns messages in reverse chronological order (newest first).

```jsonc
// 200 Response
{
  "messages": [{
    "id": "uuid",
    "conversation_id": "uuid",
    "sender_id": "uuid",
    "body": "string",
    "status": "sent|delivered",
    "created_at": "iso8601"
  }],
  "pagination": { "next_cursor": "string|null", "has_more": true }
}
```

### GET `/conversations/:id/messages/:messageId`

```jsonc
// 200 Response — single message object
```

---

## Moderation

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/users/:id/block` | Yes | Block a user |
| DELETE | `/users/:id/block` | Yes | Unblock a user |
| GET | `/users/me/blocked` | Yes | List blocked users |
| POST | `/reports` | Yes | Report a user, message, or conversation |

### POST `/users/:id/block`

```jsonc
// 204 No Content
```

### DELETE `/users/:id/block`

```jsonc
// 204 No Content
```

### GET `/users/me/blocked`

```jsonc
// 200 Response
{ "blocked": [{ "user_id": "uuid", "display_name": "string", "blocked_at": "iso8601" }] }
```

### POST `/reports`

```jsonc
// Request
{ "target_type": "user|message|conversation", "target_id": "uuid", "reason": "string" }

// 201 Response
{ "id": "uuid", "target_type": "string", "target_id": "uuid", "status": "pending", "created_at": "iso8601" }
```

---

## WebSocket

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/ws` | Yes | Upgrade to WebSocket connection |

Connect with JWT as a query parameter (`?token=`) or in the `Authorization` header. On success the server responds with `101 Switching Protocols`.

### Inbound Frames (client → server)

| Type | Payload | Description |
|------|---------|-------------|
| `ping` | — | Keepalive |
| `typing` | `{ conversation_id }` | User started typing |
| `typing_stop` | `{ conversation_id }` | User stopped typing |
| `ack` | `{ message_id }` | Message delivery acknowledgment |

### Outbound Frames (server → client)

| Type | Payload | Description |
|------|---------|-------------|
| `pong` | — | Keepalive response |
| `message` | `{ id, conversation_id, sender_id, body, created_at }` | New message |
| `typing` | `{ user_id, conversation_id }` | User is typing |
| `typing_stop` | `{ user_id, conversation_id }` | User stopped typing |
| `presence` | `{ user_id, status }` | User came online/offline |
| `delivery_ack` | `{ message_id, status }` | Delivery confirmation |

All frames are JSON-encoded: `{ "type": "<type>", "data": { ... } }`.

---

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 202 | Accepted (async processing) |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict (duplicate resource) |
| 422 | Validation Error |
| 429 | Rate Limited |
| 500 | Internal Server Error |
