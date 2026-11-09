CREATE TABLE message_mentions (
    message_id      INTEGER         NOT NULL        UNIQUE      REFERENCES messages(id) ON DELETE CASCADE,
    user_id         INTEGER         NOT NULL        UNIQUE      REFERENCES users(id) ON DELETE CASCADE,
);