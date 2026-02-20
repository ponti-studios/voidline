package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"voidline/internal/domain/conversation"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type Service struct {
	repo *sqlite.ConversationRepository
}

func NewService(repo *sqlite.ConversationRepository) *Service {
	return &Service{repo: repo}
}

type ImportOptions struct {
	SkipDuplicates bool
	FilesDir       string
}

func (s *Service) ImportTypingMind(ctx context.Context, sourcePath string, options ImportOptions) (*conversation.ImportResult, error) {
	result := &conversation.ImportResult{}
	start := time.Now()

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var input struct {
		Data struct {
			Folders        []typingMindFolder    `json:"folders"`
			UserCharacters []typingMindCharacter `json:"userCharacters"`
			UserPrompts    []typingMindPrompt    `json:"userPrompts"`
			Chats          []typingMindChat      `json:"chats"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	sourceID := generateID()
	source := &conversation.Source{
		ID:         sourceID,
		SourceType: conversation.SourceTypingMind,
		SourcePath: sourcePath,
		ImportedAt: time.Now(),
	}

	if err := s.repo.CreateSource(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	fmt.Printf("Importing TypingMind from: %s\n", sourcePath)

	for _, f := range input.Data.Folders {
		folder := s.convertTypingMindFolder(&f, sourceID)
		if err := s.repo.CreateFolder(ctx, folder); err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "folder", ID: folder.ID, Err: err,
			})
		} else {
			result.TotalFolders++
			result.Inserted++
		}
	}

	for _, c := range input.Data.UserCharacters {
		char := s.convertTypingMindCharacter(&c, sourceID)
		if err := s.repo.CreateCharacter(ctx, char); err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "character", ID: char.ID, Err: err,
			})
		} else {
			result.TotalCharacters++
			result.Inserted++
		}
	}

	for _, p := range input.Data.UserPrompts {
		prompt := s.convertTypingMindPrompt(&p, sourceID)
		if err := s.repo.CreatePrompt(ctx, prompt); err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "prompt", ID: prompt.ID, Err: err,
			})
		} else {
			result.TotalPrompts++
			result.Inserted++
		}
	}

	charIDMap := make(map[string]string)
	chars, _ := s.repo.GetCharacterBySourceID(ctx, sourceID, "")
	_ = chars

	for _, chat := range input.Data.Chats {
		exists, err := s.repo.ConversationExists(ctx, string(conversation.SourceTypingMind), chat.ID)
		if err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "conversation", ID: chat.ID, Err: err,
			})
			continue
		}

		if exists && options.SkipDuplicates {
			result.Skipped++
			continue
		}

		charID := ""
		if chat.Character != nil {
			charIDMap[chat.Character.ID] = charID
			char, err := s.repo.GetCharacterBySourceID(ctx, sourceID, chat.Character.ID)
			if err == nil && char != nil {
				charID = char.ID
			}
		}

		conv := s.convertTypingMindConversation(&chat, sourceID, charID)
		if err := s.repo.CreateConversation(ctx, conv); err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "conversation", ID: conv.ID, Err: err,
			})
		} else {
			result.TotalConversations++
			result.Inserted++
		}

		for _, msg := range chat.Messages {
			message := s.convertTypingMindMessage(&msg, conv.ID)
			if err := s.repo.CreateMessage(ctx, message); err != nil {
				result.Errors = append(result.Errors, conversation.ImportError{
					Type: "message", ID: message.ID, Err: err,
				})
			} else {
				result.TotalMessages++
				result.Inserted++
			}
		}
	}

	result.Duration = time.Since(start)
	fmt.Printf("TypingMind import complete. Conversations: %d, Messages: %d, Folders: %d, Characters: %d, Prompts: %d\n",
		result.TotalConversations, result.TotalMessages, result.TotalFolders, result.TotalCharacters, result.TotalPrompts)

	return result, nil
}

func (s *Service) ImportOpenAI(ctx context.Context, sourceDir string, options ImportOptions) (*conversation.ImportResult, error) {
	result := &conversation.ImportResult{}
	start := time.Now()

	conversationsPath := filepath.Join(sourceDir, "conversations.json")
	data, err := os.ReadFile(conversationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversations.json: %w", err)
	}

	var openAIConversations []struct {
		Title      string                       `json:"title"`
		CreateTime float64                      `json:"create_time"`
		UpdateTime float64                      `json:"update_time"`
		Mapping    map[string]openAIMappingNode `json:"mapping"`
	}
	if err := json.Unmarshal(data, &openAIConversations); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	sourceID := generateID()
	source := &conversation.Source{
		ID:         sourceID,
		SourceType: conversation.SourceOpenAI,
		SourcePath: sourceDir,
		ImportedAt: time.Now(),
	}

	if err := s.repo.CreateSource(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	userJSONPath := filepath.Join(sourceDir, "user.json")
	if data, err := os.ReadFile(userJSONPath); err == nil {
		var user struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}
		if json.Unmarshal(data, &user) == nil {
			setting := &conversation.Setting{
				Key:       "openai_user_email",
				ValueJSON: fmt.Sprintf(`"%s"`, user.Email),
				SourceID:  &sourceID,
			}
			s.repo.CreateSetting(ctx, setting)
		}
	}

	fmt.Printf("Importing OpenAI conversations from: %s\n", sourceDir)

	for _, oc := range openAIConversations {
		exists, err := s.repo.ConversationExists(ctx, string(conversation.SourceOpenAI), oc.Title)
		if err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "conversation", ID: oc.Title, Err: err,
			})
			continue
		}

		if exists && options.SkipDuplicates {
			result.Skipped++
			continue
		}

		conv := &conversation.Conversation{
			ID:                   generateID(),
			SourceID:             sourceID,
			SourceConversationID: oc.Title,
			SourceType:           conversation.SourceOpenAI,
			Title:                &oc.Title,
			CreatedAt:            floatToTime(oc.CreateTime),
			UpdatedAt:            floatToTime(oc.UpdateTime),
		}

		if err := s.repo.CreateConversation(ctx, conv); err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "conversation", ID: conv.ID, Err: err,
			})
		} else {
			result.TotalConversations++
			result.Inserted++
		}

		messages := s.extractOpenAIMessages(oc.Mapping, conv.ID)
		if err := s.repo.CreateMessageBatch(ctx, messages); err != nil {
			result.Errors = append(result.Errors, conversation.ImportError{
				Type: "messages", ID: conv.ID, Err: err,
			})
		} else {
			result.TotalMessages += len(messages)
			result.Inserted += len(messages)
		}

		dalleDir := filepath.Join(sourceDir, "dalle-generations")
		if files, err := os.ReadDir(dalleDir); err == nil {
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				srcPath := filepath.Join(dalleDir, f.Name())
				destDir := filepath.Join(options.FilesDir, "openai", conv.ID)
				destPath := filepath.Join(destDir, f.Name())

				if err := os.MkdirAll(destDir, 0755); err == nil {
					if data, err := os.ReadFile(srcPath); err == nil {
						os.WriteFile(destPath, data, 0644)
					}
				}

				file := &conversation.File{
					ID:             generateID(),
					SourceID:       sourceID,
					SourceFileID:   nil,
					FileName:       f.Name(),
					FileType:       ptr("image/webp"),
					FilePath:       &destPath,
					SourcePath:     &srcPath,
					ConversationID: &conv.ID,
				}
				if err := s.repo.CreateFile(ctx, file); err == nil {
					result.TotalFiles++
					result.Inserted++
				}
			}
		}
	}

	result.Duration = time.Since(start)
	fmt.Printf("OpenAI import complete. Conversations: %d, Messages: %d, Files: %d\n",
		result.TotalConversations, result.TotalMessages, result.TotalFiles)

	return result, nil
}

func (s *Service) convertTypingMindFolder(f *typingMindFolder, sourceID string) *conversation.Folder {
	var parentID *string
	if f.ParentID != "" {
		parentID = &f.ParentID
	}
	return &conversation.Folder{
		ID:             generateID(),
		SourceID:       sourceID,
		SourceFolderID: f.ID,
		Title:          f.Title,
		ParentID:       parentID,
		SortOrder:      f.Order,
		CreatedAt:      parseTime(f.CreatedAt),
		UpdatedAt:      parseTime(f.UpdatedAt),
		SettingsJSON:   marshalJSON(f.Settings),
	}
}

func (s *Service) convertTypingMindCharacter(c *typingMindCharacter, sourceID string) *conversation.Character {
	categories, _ := json.Marshal(c.Categories)
	return &conversation.Character{
		ID:                generateID(),
		SourceID:          sourceID,
		SourceCharacterID: c.ID,
		Name:              c.Name,
		Description:       c.Description,
		AvatarURL:         c.AvatarURL,
		Instruction:       c.Instruction,
		Categories:        string(categories),
		SettingsJSON:      marshalJSON(c),
		CreatedAt:         parseTime(c.CreatedAt),
		UpdatedAt:         parseTime(c.LastUsedAt),
	}
}

func (s *Service) convertTypingMindPrompt(p *typingMindPrompt, sourceID string) *conversation.Prompt {
	return &conversation.Prompt{
		ID:             generateID(),
		SourceID:       sourceID,
		SourcePromptID: p.ID,
		Name:           p.Name,
		Content:        p.Content,
		CreatedAt:      parseTime(p.CreatedAt),
		UpdatedAt:      parseTime(p.UpdatedAt),
	}
}

func (s *Service) convertTypingMindConversation(chat *typingMindChat, sourceID, charID string) *conversation.Conversation {
	var title string
	if chat.ChatTitle != "" {
		title = chat.ChatTitle
	} else if chat.Title != "" {
		title = chat.Title
	}

	var preview string
	if len(chat.Preview) > 500 {
		preview = chat.Preview[:500]
	} else {
		preview = chat.Preview
	}

	totalTokens := 0
	if chat.TokenUsage.TotalTokens > 0 {
		totalTokens = chat.TokenUsage.TotalTokens
	}

	var characterID *string
	if charID != "" {
		characterID = &charID
	}

	return &conversation.Conversation{
		ID:                   generateID(),
		SourceID:             sourceID,
		SourceConversationID: chat.ID,
		SourceType:           conversation.SourceTypingMind,
		Title:                &title,
		Model:                chat.Model,
		Preview:              &preview,
		TotalTokens:          totalTokens,
		CharacterID:          characterID,
		CreatedAt:            parseTime(chat.CreatedAt),
		UpdatedAt:            parseTime(chat.UpdatedAt),
		MetadataJSON:         marshalJSON(chat.ChatParams),
	}
}

func (s *Service) convertTypingMindMessage(m *typingMindMessage, convID string) *conversation.Message {
	var content *string
	if m.Content != "" {
		content = &m.Content
	}

	usageJSON := marshalJSON(m.Usage)
	tokenCount := 0
	if m.Usage != nil {
		if m.Usage.TotalTokens > 0 {
			tokenCount = m.Usage.TotalTokens
		}
	}

	return &conversation.Message{
		ID:              m.UUID,
		ConversationID:  convID,
		SourceMessageID: m.UUID,
		Role:            m.Role,
		Content:         content,
		ContentJSON:     marshalJSON(m),
		Model:           m.Model,
		TokenCount:      tokenCount,
		UsageJSON:       usageJSON,
		CreatedAt:       parseTime(m.CreatedAt),
	}
}

func (s *Service) extractOpenAIMessages(mapping map[string]openAIMappingNode, convID string) []conversation.Message {
	var messages []conversation.Message

	var traverse func(nodeID string)
	visited := make(map[string]bool)

	traverse = func(nodeID string) {
		if visited[nodeID] {
			return
		}
		visited[nodeID] = true

		node, ok := mapping[nodeID]
		if !ok || node.Message == nil {
			for _, childID := range node.Children {
				traverse(childID)
			}
			return
		}

		msg := node.Message
		if msg.Author.Role == "" {
			for _, childID := range node.Children {
				traverse(childID)
			}
			return
		}

		var content string
		if msg.Content != nil && msg.Content.Parts != nil {
			content = strings.Join(msg.Content.Parts, "\n")
		}

		var parentID *string
		if node.Parent != "" {
			parentID = &node.Parent
		}

		tokenCount := 0
		if msg.Metadata != nil {
			if promptTokens, ok := msg.Metadata["prompt_tokens"].(float64); ok {
				tokenCount += int(promptTokens)
			}
			if completionTokens, ok := msg.Metadata["completion_tokens"].(float64); ok {
				tokenCount += int(completionTokens)
			}
		}

		messages = append(messages, conversation.Message{
			ID:              msg.ID,
			ConversationID:  convID,
			SourceMessageID: msg.ID,
			ParentID:        parentID,
			Role:            msg.Author.Role,
			Content:         &content,
			ContentJSON:     marshalJSON(msg),
			Model:           nil,
			TokenCount:      tokenCount,
			UsageJSON:       marshalJSON(msg.Metadata),
			CreatedAt:       floatToTimePtr(msg.CreateTime),
		})

		for _, childID := range node.Children {
			traverse(childID)
		}
	}

	for _, node := range mapping {
		if node.Parent == "" {
			traverse(node.ID)
			break
		}
	}

	return messages
}

func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

func floatToTime(f float64) *time.Time {
	if f == 0 {
		return nil
	}
	t := time.Unix(int64(f), 0)
	return &t
}

func floatToTimePtr(f float64) *time.Time {
	if f == 0 {
		return nil
	}
	t := time.Unix(int64(f), 0)
	return &t
}

func marshalJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	data, _ := json.Marshal(v)
	return string(data)
}

func ptr[T any](v T) *T {
	return &v
}

type typingMindFolder struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	ParentID  string      `json:"parentId,omitempty"`
	Order     int         `json:"order"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	Settings  interface{} `json:"settings"`
}

type typingMindCharacter struct {
	ID          string   `json:"id"`
	Name        string   `json:"title"`
	Description *string  `json:"description"`
	AvatarURL   *string  `json:"avatarURL"`
	Instruction *string  `json:"instruction"`
	Categories  []string `json:"categories"`
	CreatedAt   string   `json:"createdAt"`
	LastUsedAt  string   `json:"lastUsedAt"`
}

type typingMindPrompt struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Content   *string `json:"content"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

type typingMindChat struct {
	ID         string                  `json:"id"`
	ChatID     string                  `json:"chatID"`
	ChatTitle  string                  `json:"chatTitle"`
	Title      string                  `json:"title"`
	Model      *string                 `json:"model"`
	Preview    string                  `json:"preview"`
	CreatedAt  string                  `json:"createdAt"`
	UpdatedAt  string                  `json:"updatedAt"`
	TokenUsage typingMindTokenUsage    `json:"tokenUsage"`
	Messages   []typingMindMessage     `json:"messages"`
	Character  *typingMindCharacterRef `json:"character"`
	ChatParams interface{}             `json:"chatParams"`
}

type typingMindCharacterRef struct {
	ID   string `json:"id"`
	Name string `json:"title"`
}

type typingMindMessage struct {
	UUID      string           `json:"uuid"`
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	Model     *string          `json:"model"`
	CreatedAt string           `json:"createdAt"`
	Usage     *typingMindUsage `json:"usage"`
	Finish    *string          `json:"finish"`
}

type typingMindTokenUsage struct {
	TotalTokens int `json:"totalTokens"`
}

type typingMindUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIMappingNode struct {
	ID       string         `json:"id"`
	Message  *openAIMessage `json:"message"`
	Parent   string         `json:"parent"`
	Children []string       `json:"children"`
}

type openAIMessage struct {
	ID         string                 `json:"id"`
	Author     openAIAuthor           `json:"author"`
	CreateTime float64                `json:"create_time"`
	UpdateTime float64                `json:"update_time"`
	Content    *openAIContent         `json:"content"`
	Status     string                 `json:"status"`
	Metadata   map[string]interface{} `json:"metadata"`
	EndTurn    *bool                  `json:"end_turn"`
}

type openAIAuthor struct {
	Role string `json:"role"`
	Name string `json:"name"`
}

type openAIContent struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}
