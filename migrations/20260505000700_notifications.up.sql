CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type text NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    read_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notifications_type_length CHECK (length(type) BETWEEN 1 AND 80),
    CONSTRAINT notifications_title_length CHECK (length(title) BETWEEN 1 AND 160),
    CONSTRAINT notifications_body_length CHECK (length(body) BETWEEN 1 AND 500),
    CONSTRAINT notifications_entity_type_length CHECK (length(entity_type) BETWEEN 1 AND 80)
);

CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, created_at DESC) WHERE read_at IS NULL;
