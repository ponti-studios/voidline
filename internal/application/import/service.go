package importservice

import (
	"context"
	"fmt"
	"time"

	"voidline/internal/domain/account"
	"voidline/internal/domain/transaction"
)

// ImportOptions contains options for the import operation
type ImportOptions struct {
	DryRun     bool   // Validate without inserting
	AutoAlias  bool   // Automatically create aliases for unknown accounts
	Force      bool   // Skip duplicate checking
	Source     string // Source identifier for import tracking
	DateFormat string // Date format string
}

// ImportError represents an error during import
type ImportError struct {
	Row  int
	Col  int
	Err  error
	Data map[string]string
}

// Error implements the error interface
func (e ImportError) Error() string {
	return fmt.Sprintf("row %d: %v", e.Row, e.Err)
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	TotalRows   int
	Inserted    int
	Skipped     int
	Errors      []ImportError
	NewAliases  []string
	NewAccounts []string
	Duration    time.Duration
}

// IsSuccess returns true if the import completed without errors
func (r ImportResult) IsSuccess() bool {
	return len(r.Errors) == 0
}

// Service handles transaction import operations
type Service struct {
	transactionRepo transaction.Repository
	accountRepo     account.Repository
	aliasManager    *account.AliasManager
	validator       *Validator
	batchSize       int
}

// NewService creates a new import service
func NewService(
	transactionRepo transaction.Repository,
	accountRepo account.Repository,
	aliasManager *account.AliasManager,
	validator *Validator,
	batchSize int,
) *Service {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &Service{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		aliasManager:    aliasManager,
		validator:       validator,
		batchSize:       batchSize,
	}
}

// ImportTransactions imports a batch of transactions
func (s *Service) ImportTransactions(
	ctx context.Context,
	transactions []transaction.Transaction,
	options ImportOptions,
) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{
		TotalRows: len(transactions),
	}

	batch := make([]transaction.Transaction, 0, s.batchSize)

	for i, tx := range transactions {
		// Validate transaction
		if err := s.validator.Validate(tx); err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row: i + 1,
				Err: err,
			})
			continue
		}

		// Normalize account name
		canonicalName, isNewAlias, err := s.aliasManager.Normalize(ctx, tx.Account)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row: i + 1,
				Err: fmt.Errorf("alias normalization failed: %w", err),
			})
			continue
		}

		if isNewAlias {
			result.NewAliases = append(result.NewAliases, tx.Account)
		}

		tx.Account = canonicalName

		// Check for duplicates (unless Force flag)
		if !options.Force {
			exists, err := s.transactionRepo.Exists(ctx, tx)
			if err != nil {
				result.Errors = append(result.Errors, ImportError{
					Row: i + 1,
					Err: fmt.Errorf("duplicate check failed: %w", err),
				})
				continue
			}
			if exists {
				result.Skipped++
				continue
			}
		}

		batch = append(batch, tx)

		// Insert batch when full
		if len(batch) >= s.batchSize && !options.DryRun {
			if err := s.insertBatch(ctx, batch); err != nil {
				return result, fmt.Errorf("batch insert failed: %w", err)
			}
			result.Inserted += len(batch)
			batch = batch[:0]
		}
	}

	// Insert remaining transactions
	if len(batch) > 0 && !options.DryRun {
		if err := s.insertBatch(ctx, batch); err != nil {
			return result, fmt.Errorf("final batch insert failed: %w", err)
		}
		result.Inserted += len(batch)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// insertBatch inserts a batch of transactions within a transaction
func (s *Service) insertBatch(ctx context.Context, transactions []transaction.Transaction) error {
	for i := range transactions {
		if err := s.transactionRepo.Create(ctx, &transactions[i]); err != nil {
			return fmt.Errorf("failed to create transaction at index %d: %w", i, err)
		}
	}
	return nil
}

// ValidateOnly validates transactions without importing them
func (s *Service) ValidateOnly(
	ctx context.Context,
	transactions []transaction.Transaction,
) (*ImportResult, error) {
	options := ImportOptions{DryRun: true}
	return s.ImportTransactions(ctx, transactions, options)
}
