package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"voidline/internal/domain/apple"
)

type AppleRepository struct {
	db *sql.DB
}

func NewAppleRepository(db *sql.DB) *AppleRepository {
	return &AppleRepository{db: db}
}

func (r *AppleRepository) EnsureTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS unified_contacts (
			id INTEGER PRIMARY KEY,
			name TEXT,
			phone TEXT,
			email TEXT,
			organization TEXT,
			source_file TEXT,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS unified_notes (
			id INTEGER PRIMARY KEY,
			title TEXT,
			content TEXT,
			folder TEXT,
			source_file TEXT,
			created_at TEXT,
			updated_at TEXT
		)`,
	}

	for _, q := range queries {
		if _, err := r.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}
	return nil
}

func (r *AppleRepository) InsertContact(ctx context.Context, contact *apple.Contact) error {
	query := `INSERT INTO unified_contacts (name, phone, email, organization, source_file, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		contact.Name,
		contact.Phone,
		contact.Email,
		contact.Organization,
		contact.SourceFile,
		time.Now().Format(time.RFC3339),
	)
	return err
}

func (r *AppleRepository) InsertNote(ctx context.Context, note *apple.Note) error {
	query := `INSERT INTO unified_notes (title, content, folder, source_file, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		note.Title,
		note.Content,
		note.Folder,
		note.SourceFile,
		note.CreatedAt,
	)
	return err
}
