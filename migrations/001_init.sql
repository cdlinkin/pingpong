CREATE TABLE users (
    id                  SERIAL          PRIMARY KEY,
    login               VARCHAR(25)     NOT NULL        CHECK(char_length(login) BETWEEN 3 AND 25) UNIQUE,
    email               VARCHAR(250)    NOT NULL        UNIQUE,
    password_hash       VARCHAR(60)     NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL        DEFAULT now()
);

CREATE TABLE chats (
    id                  SERIAL          PRIMARY KEY,
    title               VARCHAR(50),
    type                VARCHAR(50)     NOT NULL        CHECK (type IN ('direct', 'group')),
    CHECK (
        type = 'direct' 
        OR 
        title IS NOT NULL
        ),

    created_at          TIMESTAMPTZ     NOT NULL        DEFAULT now()
);

CREATE TABLE chat_members(
    user_id             INTEGER          NOT NULL        REFERENCES users(id),
    chat_id             INTEGER          NOT NULL        REFERENCES chats(id),
    role                VARCHAR(15)     NOT NULL        CHECK (role IN ('admin', 'member')),
    PRIMARY KEY (chat_id, user_id)
);

CREATE TABLE messages(
    id                  SERIAL          PRIMARY KEY,
    user_id             INTEGER         NOT NULL        REFERENCES users(id),
    chat_id             INTEGER         NOT NULL        REFERENCES chats(id),
    message             VARCHAR(500)    NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL        DEFAULT now()
);