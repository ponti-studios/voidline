package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"voidline/internal/domain/account"
)

// AliasRepository implements the account.AliasRepository interface
type AliasRepository struct {
	db *sql.DB
}

// NewAliasRepository creates a new AliasRepository
func NewAliasRepository(db *sql.DB) *AliasRepository {
	return &AliasRepository{db: db}
}

// FindByAlias retrieves a canonical name by its alias
func (r *AliasRepository) FindByAlias(ctx context.Context, alias string) (string, error) {
	query := `
		SELECT canonical_name
		FROM account_aliases
		WHERE alias = ?
	`

	var canonical string
	err := r.db.QueryRowContext(ctx, query, alias).Scan(&canonical)
	if err != nil {
		return "", err
	}

	return canonical, nil
}

// FindAll retrieves all alias mappings
func (r *AliasRepository) FindAll(ctx context.Context) ([]account.AliasMapping, error) {
	query := `
		SELECT id, alias, canonical_name, account_id, confidence_score,
		       validation_count, last_seen_at, created_at
		FROM account_aliases
		ORDER BY alias
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query aliases: %w", err)
	}
	defer rows.Close()

	return r.scanAliases(rows)
}

// Create inserts a new alias mapping
func (r *AliasRepository) Create(ctx context.Context, mapping *account.AliasMapping) error {
	query := `
		INSERT INTO account_aliases (alias, canonical_name, account_id, confidence_score, last_seen_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(alias) DO UPDATE SET
			canonical_name = excluded.canonical_name,
			confidence_score = excluded.confidence_score,
			last_seen_at = datetime('now')
	`

	var accountID interface{}
	if mapping.AccountID != nil {
		accountID = *mapping.AccountID
	} else {
		accountID = nil
	}

	result, err := r.db.ExecContext(ctx, query,
		mapping.Alias, mapping.CanonicalName, accountID, mapping.ConfidenceScore,
	)
	if err != nil {
		return fmt.Errorf("failed to create alias: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	mapping.ID = int(id)
	return nil
}

// Update updates an existing alias mapping
func (r *AliasRepository) Update(ctx context.Context, mapping *account.AliasMapping) error {
	query := `
		UPDATE account_aliases
		SET alias = ?, canonical_name = ?, account_id = ?,
		    confidence_score = ?, validation_count = ?
		WHERE id = ?
	`

	var accountID interface{}
	if mapping.AccountID != nil {
		accountID = *mapping.AccountID
	} else {
		accountID = nil
	}

	_, err := r.db.ExecContext(ctx, query,
		mapping.Alias, mapping.CanonicalName, accountID,
		mapping.ConfidenceScore, mapping.ValidationCount, mapping.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update alias: %w", err)
	}

	return nil
}

// Delete removes an alias by ID
func (r *AliasRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM account_aliases WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete alias: %w", err)
	}
	return nil
}

// IncrementValidationCount increments the validation count for an alias
func (r *AliasRepository) IncrementValidationCount(ctx context.Context, alias string) error {
	query := `
		UPDATE account_aliases
		SET validation_count = validation_count + 1,
		    last_seen_at = datetime('now')
		WHERE alias = ?
	`

	_, err := r.db.ExecContext(ctx, query, alias)
	if err != nil {
		return fmt.Errorf("failed to increment validation count: %w", err)
	}

	return nil
}

// GetUnvalidated returns aliases with low confidence scores
func (r *AliasRepository) GetUnvalidated(ctx context.Context, threshold float64) ([]account.AliasMapping, error) {
	query := `
		SELECT id, alias, canonical_name, account_id, confidence_score,
		       validation_count, last_seen_at, created_at
		FROM account_aliases
		WHERE confidence_score <= ?
		ORDER BY confidence_score ASC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query unvalidated aliases: %w", err)
	}
	defer rows.Close()

	return r.scanAliases(rows)
}

// scanAliases scans alias rows into a slice
func (r *AliasRepository) scanAliases(rows *sql.Rows) ([]account.AliasMapping, error) {
	var mappings []account.AliasMapping

	for rows.Next() {
		var m account.AliasMapping
		var accountID sql.NullInt64
		var lastSeenAt, createdAt sql.NullString

		err := rows.Scan(
			&m.ID, &m.Alias, &m.CanonicalName, &accountID,
			&m.ConfidenceScore, &m.ValidationCount, &lastSeenAt, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alias: %w", err)
		}

		if accountID.Valid {
			id := int(accountID.Int64)
			m.AccountID = &id
		}
		if lastSeenAt.Valid {
			m.LastSeenAt = lastSeenAt.String
		}
		if createdAt.Valid {
			m.CreatedAt = createdAt.String
		}

		mappings = append(mappings, m)
	}

	return mappings, rows.Err()
}
