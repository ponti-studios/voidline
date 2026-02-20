package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"voidline/internal/domain/timeutil"
	"voidline/internal/domain/transaction"
)

// TransactionRepository implements the transaction.Repository interface
type TransactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository creates a new TransactionRepository
func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// FindByID retrieves a transaction by its ID
func (r *TransactionRepository) FindByID(ctx context.Context, id int) (*transaction.Transaction, error) {
	query := `
		SELECT id, date, name, amount, status, category, parent_category, 
		       excluded, tags, type, account, account_mask, note, recurring, 
		       created_at, updated_at
		FROM finance_transactions
		WHERE id = ?
	`

	var tx transaction.Transaction
	var dateStr, createdAt, updatedAt string
	var excluded int
	var recurring sql.NullInt64
	var tags, txType, account, accountMask, note sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tx.ID, &dateStr, &tx.Name, &tx.Amount, &tx.Status,
		&tx.Category, &tx.ParentCategory, &excluded, &tags,
		&txType, &account, &accountMask, &note, &recurring,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}

	tx.Date, _ = timeutil.ParseDate(dateStr)
	tx.Excluded = excluded == 1
	tx.Tags = tags.String
	tx.Type = txType.String
	tx.Account = account.String
	tx.AccountMask = accountMask.String
	tx.Note = note.String
	tx.Recurring = recurring.Int64 == 1

	return &tx, nil
}

// FindByFilter retrieves paginated transactions matching the filter
func (r *TransactionRepository) FindByFilter(ctx context.Context, filter transaction.Filter) (*transaction.PaginatedResult, error) {
	where, args := r.buildWhereClause(filter)

	// Get total count
	countQuery := "SELECT COUNT(*) FROM finance_transactions " + where
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Get paginated results
	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`
		SELECT id, date, name, amount, status, category, parent_category, 
		       excluded, tags, type, account, account_mask, note, recurring,
		       created_at, updated_at
		FROM finance_transactions
		%s
		ORDER BY date DESC, id DESC
		LIMIT ? OFFSET ?
	`, where)

	args = append(args, filter.PerPage, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var items []transaction.Transaction
	for rows.Next() {
		var tx transaction.Transaction
		var dateStr, createdAt, updatedAt string
		var excluded int
		var recurring sql.NullInt64
		var tags, txType, account, accountMask, note sql.NullString

		err := rows.Scan(
			&tx.ID, &dateStr, &tx.Name, &tx.Amount, &tx.Status,
			&tx.Category, &tx.ParentCategory, &excluded, &tags,
			&txType, &account, &accountMask, &note, &recurring,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		tx.Date, _ = timeutil.ParseDate(dateStr)
		tx.Excluded = excluded == 1
		tx.Tags = tags.String
		tx.Type = txType.String
		tx.Account = account.String
		tx.AccountMask = accountMask.String
		tx.Note = note.String
		tx.Recurring = recurring.Int64 == 1

		items = append(items, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	result := transaction.NewPaginatedResult(items, totalCount, filter.Page, filter.PerPage)
	return &result, nil
}

// FindAll retrieves all transactions (use with caution)
func (r *TransactionRepository) FindAll(ctx context.Context) ([]transaction.Transaction, error) {
	filter := transaction.NewFilter().WithPerPage(100000) // Large number to get all
	result, err := r.FindByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Create inserts a new transaction
func (r *TransactionRepository) Create(ctx context.Context, tx *transaction.Transaction) error {
	query := `
		INSERT INTO finance_transactions (
			date, name, amount, status, category, parent_category, 
			excluded, tags, type, account, account_mask, note, recurring,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`

	excluded := 0
	if tx.Excluded {
		excluded = 1
	}

	recurring := 0
	if tx.Recurring {
		recurring = 1
	}

	result, err := r.db.ExecContext(ctx, query,
		tx.Date.String(), tx.Name, tx.Amount, tx.Status,
		tx.Category, tx.ParentCategory, excluded, tx.Tags,
		tx.Type, tx.Account, tx.AccountMask, tx.Note, recurring,
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	tx.ID = int(id)
	return nil
}

// CreateBatch inserts multiple transactions in a batch
func (r *TransactionRepository) CreateBatch(ctx context.Context, transactions []transaction.Transaction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i := range transactions {
		if err := r.Create(ctx, &transactions[i]); err != nil {
			return fmt.Errorf("failed to create transaction at index %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// Update updates an existing transaction
func (r *TransactionRepository) Update(ctx context.Context, tx *transaction.Transaction) error {
	query := `
		UPDATE finance_transactions
		SET date = ?, name = ?, amount = ?, status = ?, category = ?,
		    parent_category = ?, excluded = ?, tags = ?, type = ?,
		    account = ?, account_mask = ?, note = ?, recurring = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`

	excluded := 0
	if tx.Excluded {
		excluded = 1
	}

	recurring := 0
	if tx.Recurring {
		recurring = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		tx.Date.String(), tx.Name, tx.Amount, tx.Status,
		tx.Category, tx.ParentCategory, excluded, tx.Tags,
		tx.Type, tx.Account, tx.AccountMask, tx.Note, recurring,
		tx.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

// Delete removes a transaction by ID
func (r *TransactionRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM finance_transactions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}
	return nil
}

// Exists checks if a transaction exists (for deduplication)
func (r *TransactionRepository) Exists(ctx context.Context, tx transaction.Transaction) (bool, error) {
	query := `
		SELECT 1 FROM finance_transactions
		WHERE date = ? AND name = ? AND account = ? AND ABS(amount - ?) < 0.01
		LIMIT 1
	`

	var exists int
	err := r.db.QueryRowContext(ctx, query,
		tx.Date.String(), tx.Name, tx.Account, tx.Amount,
	).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return true, nil
}

// Count returns the total count of transactions matching the filter
func (r *TransactionRepository) Count(ctx context.Context, filter transaction.Filter) (int, error) {
	where, args := r.buildWhereClause(filter)
	query := "SELECT COUNT(*) FROM finance_transactions " + where

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

// GetCategories returns all unique categories
func (r *TransactionRepository) GetCategories(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT category FROM finance_transactions
		WHERE category IS NOT NULL AND category != ''
		ORDER BY category
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

// GetAccounts returns all unique accounts
func (r *TransactionRepository) GetAccounts(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT account FROM finance_transactions
		WHERE account IS NOT NULL AND account != ''
		ORDER BY account
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	defer rows.Close()

	var accounts []string
	for rows.Next() {
		var acc string
		if err := rows.Scan(&acc); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, acc)
	}

	return accounts, rows.Err()
}

// GetDateRange returns the min and max dates in the database
func (r *TransactionRepository) GetDateRange(ctx context.Context) (min, max string, err error) {
	query := `
		SELECT MIN(date), MAX(date) FROM finance_transactions
		WHERE excluded = 0
	`

	if err := r.db.QueryRowContext(ctx, query).Scan(&min, &max); err != nil {
		return "", "", fmt.Errorf("failed to get date range: %w", err)
	}

	return min, max, nil
}

// buildWhereClause builds the WHERE clause and arguments for filtering
func (r *TransactionRepository) buildWhereClause(filter transaction.Filter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.StartDate != nil && !filter.StartDate.IsZero() {
		conditions = append(conditions, "date >= ?")
		args = append(args, filter.StartDate.String())
	}
	if filter.EndDate != nil && !filter.EndDate.IsZero() {
		conditions = append(conditions, "date <= ?")
		args = append(args, filter.EndDate.String())
	}
	if len(filter.Accounts) > 0 {
		placeholders := make([]string, len(filter.Accounts))
		for i := range filter.Accounts {
			placeholders[i] = "?"
			args = append(args, filter.Accounts[i])
		}
		conditions = append(conditions, fmt.Sprintf("account IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(filter.Categories) > 0 {
		placeholders := make([]string, len(filter.Categories))
		for i := range filter.Categories {
			placeholders[i] = "?"
			args = append(args, filter.Categories[i])
		}
		conditions = append(conditions, fmt.Sprintf("category IN (%s)", strings.Join(placeholders, ",")))
	}
	if filter.MinAmount != nil {
		conditions = append(conditions, "ABS(amount) >= ?")
		args = append(args, *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		conditions = append(conditions, "ABS(amount) <= ?")
		args = append(args, *filter.MaxAmount)
	}
	if filter.SearchQuery != "" {
		conditions = append(conditions, "(name LIKE ? OR note LIKE ?)")
		likePattern := "%" + filter.SearchQuery + "%"
		args = append(args, likePattern, likePattern)
	}
	if filter.Recurring != nil {
		conditions = append(conditions, "recurring = ?")
		if *filter.Recurring {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if filter.Excluded != nil {
		conditions = append(conditions, "excluded = ?")
		if *filter.Excluded {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}
