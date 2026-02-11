CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID        NOT NULL REFERENCES users(id),
    body            TEXT        NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'sent',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation_id ON messages (conversation_id, created_at DESC);

CREATE TABLE message_deliveries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id   UUID        NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    delivered_at TIMESTAMPTZ,
    read_at      TIMESTAMPTZ,

    CONSTRAINT unique_delivery UNIQUE (message_id, user_id)
);

CREATE INDEX idx_md_message_id ON message_deliveries (message_id);
CREATE INDEX idx_md_user_id ON message_deliveries (user_id);
