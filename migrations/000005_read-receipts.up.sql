ALTER TABLE chat_members 
ADD COLUMN last_read_message_id  INTEGER NOT NULL DEFAULT 0;