package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"voidline/internal/domain/conversation"
)

type ConversationRepository struct {
	db *sql.DB
}

func NewConversationRepository(db *sql.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) CreateSource(ctx context.Context, source *conversation.Source) error {
	query := `
		INSERT INTO conversation_sources (id, source_type, source_path, imported_at, metadata_json)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		source.ID,
		source.SourceType,
		source.SourcePath,
		source.ImportedAt.Format(time.RFC3339),
		source.MetadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}
	return nil
}

func (r *ConversationRepository) SourceExists(ctx context.Context, sourceType, sourcePath string) (bool, error) {
	query := `SELECT COUNT(*) FROM conversation_sources WHERE source_type = ? AND source_path = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, sourceType, sourcePath).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check source exists: %w", err)
	}
	return count > 0, nil
}

func (r *ConversationRepository) CreateFolder(ctx context.Context, folder *conversation.Folder) error {
	query := `
		INSERT INTO conversation_folders (
			id, source_id, source_folder_id, title, parent_id, sort_order,
			created_at, updated_at, settings_json, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		folder.ID,
		folder.SourceID,
		folder.SourceFolderID,
		folder.Title,
		folder.ParentID,
		folder.SortOrder,
		nullableTime(folder.CreatedAt),
		nullableTime(folder.UpdatedAt),
		folder.SettingsJSON,
		folder.MetadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}
	return nil
}

func (r *ConversationRepository) GetFolderBySourceID(ctx context.Context, sourceID, sourceFolderID string) (*conversation.Folder, error) {
	query := `SELECT id, source_id, source_folder_id, title, parent_id, sort_order,
		created_at, updated_at, settings_json, metadata_json
		FROM conversation_folders WHERE source_id = ? AND source_folder_id = ?`
	var folder conversation.Folder
	var parentID, createdAt, updatedAt sql.NullString
	err := r.db.QueryRowContext(ctx, query, sourceID, sourceFolderID).Scan(
		&folder.ID, &folder.SourceID, &folder.SourceFolderID, &folder.Title,
		&parentID, &folder.SortOrder, &createdAt, &updatedAt,
		&folder.SettingsJSON, &folder.MetadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}
	if parentID.Valid {
		folder.ParentID = &parentID.String
	}
	folder.CreatedAt = parseTime(createdAt)
	folder.UpdatedAt = parseTime(updatedAt)
	return &folder, nil
}

func (r *ConversationRepository) CreateCharacter(ctx context.Context, char *conversation.Character) error {
	query := `
		INSERT INTO conversation_characters (
			id, source_id, source_character_id, name, description, avatar_url,
			instruction, categories, settings_json, created_at, updated_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		char.ID,
		char.SourceID,
		char.SourceCharacterID,
		char.Name,
		char.Description,
		char.AvatarURL,
		char.Instruction,
		char.Categories,
		char.SettingsJSON,
		nullableTime(char.CreatedAt),
		nullableTime(char.UpdatedAt),
		char.MetadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create character: %w", err)
	}
	return nil
}

func (r *ConversationRepository) GetCharacterBySourceID(ctx context.Context, sourceID, sourceCharID string) (*conversation.Character, error) {
	query := `SELECT id, source_id, source_character_id, name, description, avatar_url,
		instruction, categories, settings_json, created_at, updated_at, metadata_json
		FROM conversation_characters WHERE source_id = ? AND source_character_id = ?`
	var char conversation.Character
	var description, avatarURL, instruction, createdAt, updatedAt sql.NullString
	err := r.db.QueryRowContext(ctx, query, sourceID, sourceCharID).Scan(
		&char.ID, &char.SourceID, &char.SourceCharacterID, &char.Name,
		&description, &avatarURL, &instruction, &char.Categories,
		&char.SettingsJSON, &createdAt, &updatedAt, &char.MetadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get character: %w", err)
	}
	if description.Valid {
		char.Description = &description.String
	}
	if avatarURL.Valid {
		char.AvatarURL = &avatarURL.String
	}
	if instruction.Valid {
		char.Instruction = &instruction.String
	}
	char.CreatedAt = parseTime(createdAt)
	char.UpdatedAt = parseTime(updatedAt)
	return &char, nil
}

func (r *ConversationRepository) CreatePrompt(ctx context.Context, prompt *conversation.Prompt) error {
	query := `
		INSERT INTO conversation_prompts (
			id, source_id, source_prompt_id, name, content, created_at, updated_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		prompt.ID,
		prompt.SourceID,
		prompt.SourcePromptID,
		prompt.Name,
		prompt.Content,
		nullableTime(prompt.CreatedAt),
		nullableTime(prompt.UpdatedAt),
		prompt.MetadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create prompt: %w", err)
	}
	return nil
}

func (r *ConversationRepository) CreateSetting(ctx context.Context, setting *conversation.Setting) error {
	query := `
		INSERT OR REPLACE INTO conversation_settings (key, value_json, source_id)
		VALUES (?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, setting.Key, setting.ValueJSON, setting.SourceID)
	if err != nil {
		return fmt.Errorf("failed to create setting: %w", err)
	}
	return nil
}

func (r *ConversationRepository) ConversationExists(ctx context.Context, sourceType, sourceConvID string) (bool, error) {
	query := `SELECT COUNT(*) FROM conversations WHERE source_type = ? AND source_conversation_id = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, sourceType, sourceConvID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check conversation exists: %w", err)
	}
	return count > 0, nil
}

func (r *ConversationRepository) CreateConversation(ctx context.Context, conv *conversation.Conversation) error {
	query := `
		INSERT INTO conversations (
			id, source_id, source_conversation_id, source_type, title, model, preview,
			total_tokens, folder_id, character_id, created_at, updated_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		conv.ID,
		conv.SourceID,
		conv.SourceConversationID,
		conv.SourceType,
		conv.Title,
		conv.Model,
		conv.Preview,
		conv.TotalTokens,
		conv.FolderID,
		conv.CharacterID,
		nullableTime(conv.CreatedAt),
		nullableTime(conv.UpdatedAt),
		conv.MetadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) CreateMessage(ctx context.Context, msg *conversation.Message) error {
	query := `
		INSERT INTO conversation_messages (
			id, conversation_id, source_message_id, parent_id, role, content,
			content_json, model, token_count, usage_json, created_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		msg.ID,
		msg.ConversationID,
		msg.SourceMessageID,
		msg.ParentID,
		msg.Role,
		msg.Content,
		msg.ContentJSON,
		msg.Model,
		msg.TokenCount,
		msg.UsageJSON,
		nullableTime(msg.CreatedAt),
		msg.MetadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	return nil
}

func (r *ConversationRepository) CreateMessageBatch(ctx context.Context, msgs []conversation.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO conversation_messages (
			id, conversation_id, source_message_id, parent_id, role, content,
			content_json, model, token_count, usage_json, created_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, msg := range msgs {
		_, err := stmt.ExecContext(ctx,
			msg.ID,
			msg.ConversationID,
			msg.SourceMessageID,
			msg.ParentID,
			msg.Role,
			msg.Content,
			msg.ContentJSON,
			msg.Model,
			msg.TokenCount,
			msg.UsageJSON,
			nullableTime(msg.CreatedAt),
			msg.MetadataJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert message: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *ConversationRepository) CreateFile(ctx context.Context, file *conversation.File) error {
	query := `
		INSERT INTO conversation_files (
			id, source_id, source_file_id, file_name, file_type, file_path,
			source_path, content_text, metadata_json, conversation_id, character_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		file.ID,
		file.SourceID,
		file.SourceFileID,
		file.FileName,
		file.FileType,
		file.FilePath,
		file.SourcePath,
		file.ContentText,
		file.MetadataJSON,
		file.ConversationID,
		file.CharacterID,
		nullableTime(file.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

func (r *ConversationRepository) GetConversationBySourceID(ctx context.Context, sourceType, sourceConvID string) (*conversation.Conversation, error) {
	query := `SELECT id, source_id, source_conversation_id, source_type, title, model, preview,
		total_tokens, folder_id, character_id, created_at, updated_at, metadata_json
		FROM conversations WHERE source_type = ? AND source_conversation_id = ?`
	var conv conversation.Conversation
	var title, model, preview, createdAt, updatedAt sql.NullString
	err := r.db.QueryRowContext(ctx, query, sourceType, sourceConvID).Scan(
		&conv.ID, &conv.SourceID, &conv.SourceConversationID, &conv.SourceType,
		&title, &model, &preview, &conv.TotalTokens, &conv.FolderID,
		&conv.CharacterID, &createdAt, &updatedAt, &conv.MetadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if title.Valid {
		conv.Title = &title.String
	}
	if model.Valid {
		conv.Model = &model.String
	}
	if preview.Valid {
		conv.Preview = &preview.String
	}
	conv.CreatedAt = parseTime(createdAt)
	conv.UpdatedAt = parseTime(updatedAt)
	return &conv, nil
}

func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func parseTime(s sql.NullString) *time.Time {
	if !s.Valid {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s.String)
	if err != nil {
		return nil
	}
	return &t
}
