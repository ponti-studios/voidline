-- +goose Up
-- Conversations Domain Migration
-- Imports from TypingMind and OpenAI conversation exports

-- Source tracking
CREATE TABLE IF NOT EXISTS conversation_sources (
    id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_path TEXT NOT NULL,
    imported_at TEXT NOT NULL,
    metadata_json TEXT
);

-- Folders (TypingMind)
CREATE TABLE IF NOT EXISTS conversation_folders (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES conversation_sources(id),
    source_folder_id TEXT,
    title TEXT NOT NULL,
    parent_id TEXT,
    sort_order INTEGER DEFAULT 0,
    created_at TEXT,
    updated_at TEXT,
    settings_json TEXT,
    metadata_json TEXT
);

-- Characters (TypingMind custom AI personas)
CREATE TABLE IF NOT EXISTS conversation_characters (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES conversation_sources(id),
    source_character_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    avatar_url TEXT,
    instruction TEXT,
    categories TEXT,
    settings_json TEXT,
    created_at TEXT,
    updated_at TEXT,
    metadata_json TEXT
);

-- Prompts (TypingMind saved prompts)
CREATE TABLE IF NOT EXISTS conversation_prompts (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES conversation_sources(id),
    source_prompt_id TEXT,
    name TEXT NOT NULL,
    content TEXT,
    created_at TEXT,
    updated_at TEXT,
    metadata_json TEXT
);

-- User settings (key-value store)
CREATE TABLE IF NOT EXISTS conversation_settings (
    key TEXT PRIMARY KEY,
    value_json TEXT,
    source_id TEXT REFERENCES conversation_sources(id)
);

-- Conversations
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES conversation_sources(id),
    source_conversation_id TEXT NOT NULL,
    source_type TEXT NOT NULL,
    title TEXT,
    model TEXT,
    preview TEXT,
    total_tokens INTEGER DEFAULT 0,
    folder_id TEXT REFERENCES conversation_folders(id),
    character_id TEXT REFERENCES conversation_characters(id),
    created_at TEXT,
    updated_at TEXT,
    metadata_json TEXT,
    UNIQUE(source_type, source_conversation_id)
);

-- Messages
CREATE TABLE IF NOT EXISTS conversation_messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    source_message_id TEXT NOT NULL,
    parent_id TEXT,
    role TEXT NOT NULL,
    content TEXT,
    content_json TEXT,
    model TEXT,
    token_count INTEGER DEFAULT 0,
    usage_json TEXT,
    created_at TEXT,
    metadata_json TEXT,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON conversation_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_parent ON conversation_messages(parent_id);
CREATE INDEX IF NOT EXISTS idx_conversations_source ON conversations(source_id);

-- Files references (uploaded files/images)
CREATE TABLE IF NOT EXISTS conversation_files (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES conversation_sources(id),
    source_file_id TEXT,
    file_name TEXT NOT NULL,
    file_type TEXT,
    file_path TEXT,
    source_path TEXT,
    content_text TEXT,
    metadata_json TEXT,
    conversation_id TEXT REFERENCES conversations(id),
    character_id TEXT REFERENCES conversation_characters(id),
    created_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_files_conversation ON conversation_files(conversation_id);
CREATE INDEX IF NOT EXISTS idx_files_character ON conversation_files(character_id);

-- +goose Down
DROP TABLE IF EXISTS conversation_files;
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS conversation_settings;
DROP TABLE IF EXISTS conversation_prompts;
DROP TABLE IF EXISTS conversation_characters;
DROP TABLE IF EXISTS conversation_folders;
DROP TABLE IF EXISTS conversation_sources;
