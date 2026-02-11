---
title: Chat App Plan
tags: [project, plan, chat, c4]
status: draft
---

## Vision

Build a small chat application that supports 1:1 and group messaging with real-time deliver.

## Goal

- Users can sign in/up, start a conversation, and exchange messaging with real-time delivery
- Messages persist and load correctly across devices/sessions.
- Basic safety + abuse controls (rate limiting, reporting/blocking) exist.
- Reliable UX under normal usage (low latency, minimal message loss/duplication).

## Scope

- Authentication (email/password; optional OAuth later)
- User profiles (name, avatar)
- Conversations:
  - Create 1:1 or group
  - Add/remove participants (group)
- Messaging:
  - Send/receive text messages
  - Load message history (pagination)
  - Delivery acknowledgment (at least “delivered”)
- Realtime:
  - WebSocket connection
  - Presence: online/offline (basic)
  - Typing indicator (optional for MVP)
- Notifications:
  - Push notifications (optional for MVP; can be stubbed)
- Moderation:
  - Block user
  - Report conversation/message (basic)
- Observability:
  - Request logs + error tracking
  - Basic metrics (latency, error rates)
- Attachments (images/files) + media processing
- Read receipts per-user
- Message edits/deletes
- Search
- Reactions
- E2EE

## Architecture

![[docs/diagrams/architectures.md]]

## Workflows

![[docs/diagrams/workflows.md]]

## Schema

![[docs/diagrams/schema.md]]
