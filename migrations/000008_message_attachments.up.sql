CREATE TABLE message_attachments (
    id                  SERIAL          PRIMARY KEY,
    message_id          INTEGER         NOT NULL        UNIQUE      REFERENCES messages(id) ON DELETE CASCADE,
    file_name           VARCHAR(255)    NOT NULL,
    content_type        VARCHAR(100)    NOT NULL,
    size                BIGINT          NOT NULL        CHECK (size > 0),
    storage_key         VARCHAR(64)     NOT NULL        UNIQUE,
    created_at          TIMESTAMPTZ     NOT NULL        DEFAULT now()
);
