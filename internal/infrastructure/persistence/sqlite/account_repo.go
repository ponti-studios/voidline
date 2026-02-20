package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"voidline/internal/domain/account"
)

// AccountRepository implements the account.Repository interface
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new AccountRepository
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// FindByID retrieves an account by its ID
func (r *AccountRepository) FindByID(ctx context.Context, id int) (*account.Account, error) {
	query := `
		SELECT id, name, type, credit_limit, active
		FROM financial_accounts
		WHERE id = ?
	`

	var acc account.Account
	var creditLimit sql.NullFloat64
	var active int

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&acc.ID, &acc.Name, &acc.Type, &creditLimit, &active,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	if creditLimit.Valid {
		acc.CreditLimit = &creditLimit.Float64
	}
	acc.IsActive = active == 1
	acc.CanonicalName = acc.Name
	acc.Currency = "USD"

	return &acc, nil
}

// FindByName retrieves an account by its canonical name
func (r *AccountRepository) FindByName(ctx context.Context, name string) (*account.Account, error) {
	query := `
		SELECT id, name, type, credit_limit, active
		FROM financial_accounts
		WHERE name = ?
	`

	var acc account.Account
	var creditLimit sql.NullFloat64
	var active int

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&acc.ID, &acc.Name, &acc.Type, &creditLimit, &active,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	if creditLimit.Valid {
		acc.CreditLimit = &creditLimit.Float64
	}
	acc.IsActive = active == 1
	acc.CanonicalName = acc.Name
	acc.Currency = "USD"

	return &acc, nil
}

// FindAll retrieves all accounts
func (r *AccountRepository) FindAll(ctx context.Context) ([]account.Account, error) {
	query := `
		SELECT id, name, type, credit_limit, active
		FROM financial_accounts
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}
	defer rows.Close()

	return r.scanAccounts(rows)
}

// FindActive retrieves only active accounts
func (r *AccountRepository) FindActive(ctx context.Context) ([]account.Account, error) {
	query := `
		SELECT id, name, type, credit_limit, active
		FROM financial_accounts
		WHERE active = 1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active accounts: %w", err)
	}
	defer rows.Close()

	return r.scanAccounts(rows)
}

// Create inserts a new account
func (r *AccountRepository) Create(ctx context.Context, acc *account.Account) error {
	query := `
		INSERT INTO financial_accounts (name, type, credit_limit, active)
		VALUES (?, ?, ?, ?)
	`

	var creditLimit interface{}
	if acc.CreditLimit != nil {
		creditLimit = *acc.CreditLimit
	} else {
		creditLimit = nil
	}

	active := 0
	if acc.IsActive {
		active = 1
	}

	result, err := r.db.ExecContext(ctx, query,
		acc.Name, acc.Type, creditLimit, active,
	)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	acc.ID = int(id)
	return nil
}

// Update updates an existing account
func (r *AccountRepository) Update(ctx context.Context, acc *account.Account) error {
	query := `
		UPDATE financial_accounts
		SET name = ?, type = ?, credit_limit = ?, active = ?
		WHERE id = ?
	`

	var creditLimit interface{}
	if acc.CreditLimit != nil {
		creditLimit = *acc.CreditLimit
	} else {
		creditLimit = nil
	}

	active := 0
	if acc.IsActive {
		active = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		acc.Name, acc.Type, creditLimit, active, acc.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	return nil
}

// Delete removes an account by ID
func (r *AccountRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM financial_accounts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

// GetBalances returns current balances for all accounts
func (r *AccountRepository) GetBalances(ctx context.Context) (map[string]float64, error) {
	query := `
		SELECT account, SUM(amount) as balance
		FROM finance_transactions
		WHERE excluded = 0
		GROUP BY account
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get balances: %w", err)
	}
	defer rows.Close()

	balances := make(map[string]float64)
	for rows.Next() {
		var account string
		var balance float64
		if err := rows.Scan(&account, &balance); err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}
		balances[account] = balance
	}

	return balances, rows.Err()
}

// UpdateBalance updates the current balance for an account
func (r *AccountRepository) UpdateBalance(ctx context.Context, name string, balance float64) error {
	// Note: This is a virtual operation since balance is computed from transactions
	// In a real implementation, you might store cached balances
	return nil
}

// scanAccounts scans account rows into a slice
func (r *AccountRepository) scanAccounts(rows *sql.Rows) ([]account.Account, error) {
	var accounts []account.Account

	for rows.Next() {
		var acc account.Account
		var creditLimit sql.NullFloat64
		var active int

		err := rows.Scan(&acc.ID, &acc.Name, &acc.Type, &creditLimit, &active)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		if creditLimit.Valid {
			acc.CreditLimit = &creditLimit.Float64
		}
		acc.IsActive = active == 1
		acc.CanonicalName = acc.Name
		acc.Currency = "USD"

		accounts = append(accounts, acc)
	}

	return accounts, rows.Err()
}
