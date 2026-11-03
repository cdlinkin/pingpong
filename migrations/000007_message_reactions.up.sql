CREATE TABLE message_reactions (
    message_id INTEGER     NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    INTEGER     NOT NULL REFERENCES users(id),
    emoji      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, user_id)
);