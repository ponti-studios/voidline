package services

import (
	"context"
	"fmt"
	"time"

	"voidline/internal/application/errors"
	"voidline/internal/application/validation"
	"voidline/internal/domain/timeutil"
	"voidline/internal/domain/transaction"
)

type FinanceTransactionsService struct {
	transactionRepo transaction.Repository
}

func NewFinanceTransactionsService(transactionRepo transaction.Repository) *FinanceTransactionsService {
	return &FinanceTransactionsService{transactionRepo: transactionRepo}
}

type GetTransactionsOptions struct {
	Account   string
	Category  string
	StartDate string
	EndDate   string
	Page      int
	PerPage   int
}

type FinanceTransactionDTO struct {
	ID       string  `json:"id"`
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Account  string  `json:"account"`
	Category string  `json:"category"`
	Payee    string  `json:"payee"`
	Notes    string  `json:"notes"`
}

type CreateTransactionInput struct {
	Date     string  `json:"date" validate:"required"`
	Name     string  `json:"name" validate:"required"`
	Amount   float64 `json:"amount" validate:"required"`
	Account  string  `json:"account" validate:"required"`
	Category string  `json:"category"`
	Note     string  `json:"note"`
}

type CreateTransactionsBatchInput struct {
	Transactions   []CreateTransactionInput `json:"transactions"`
	SkipDuplicates bool                     `json:"skipDuplicates"`
}

type UpdateTransactionInput struct {
	ID       string  `json:"id" validate:"required"`
	Date     string  `json:"date"`
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	Account  string  `json:"account"`
	Category string  `json:"category"`
	Note     string  `json:"note"`
}

type BatchCreateResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

func (s *FinanceTransactionsService) GetTransactions(ctx context.Context, opts GetTransactionsOptions) ([]FinanceTransactionDTO, error) {
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	filter := transaction.NewFilter().
		WithPage(opts.Page).
		WithPerPage(opts.PerPage)

	if opts.Account != "" {
		filter.Accounts = []string{opts.Account}
	}
	if opts.Category != "" {
		filter.Categories = []string{opts.Category}
	}
	if opts.StartDate != "" {
		date, err := timeutil.ParseDate(opts.StartDate)
		if err != nil {
			return nil, err
		}
		filter.StartDate = &date
	}
	if opts.EndDate != "" {
		date, err := timeutil.ParseDate(opts.EndDate)
		if err != nil {
			return nil, err
		}
		filter.EndDate = &date
	}

	result, err := s.transactionRepo.FindByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	dtos := make([]FinanceTransactionDTO, len(result.Items))
	for i, tx := range result.Items {
		dtos[i] = FinanceTransactionDTO{
			ID:       fmt.Sprintf("%d", tx.ID),
			Date:     tx.Date.String(),
			Amount:   tx.Amount,
			Account:  tx.Account,
			Category: tx.Category,
			Payee:    tx.Name,
			Notes:    tx.Note,
		}
	}

	return dtos, nil
}

func (s *FinanceTransactionsService) CreateTransaction(ctx context.Context, input CreateTransactionInput) (FinanceTransactionDTO, error) {
	var errs validation.ValidationErrors

	if err := validation.ValidateRequired("date", input.Date); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateRequired("name", input.Name); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateRequired("account", input.Account); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateDate("date", input.Date); err != nil {
		errs = append(errs, *err)
	}
	if input.Amount == 0 {
		errs = append(errs, validation.ValidationError{Field: "amount", Message: "must be non-zero"})
	}

	if errs.HasErrors() {
		return FinanceTransactionDTO{}, errs
	}

	txDate, err := timeutil.ParseDate(input.Date)
	if err != nil {
		return FinanceTransactionDTO{}, fmt.Errorf("invalid date format: %w", err)
	}

	tx := &transaction.Transaction{
		Date:     txDate,
		Name:     input.Name,
		Amount:   input.Amount,
		Account:  input.Account,
		Category: input.Category,
		Note:     input.Note,
		Status:   "posted",
	}

	err = s.transactionRepo.Create(ctx, tx)
	if err != nil {
		return FinanceTransactionDTO{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	return FinanceTransactionDTO{
		ID:       fmt.Sprintf("%d", tx.ID),
		Date:     tx.Date.String(),
		Amount:   tx.Amount,
		Account:  tx.Account,
		Category: tx.Category,
		Payee:    tx.Name,
		Notes:    tx.Note,
	}, nil
}

func (s *FinanceTransactionsService) CreateTransactionsBatch(ctx context.Context, input CreateTransactionsBatchInput) (BatchCreateResult, error) {
	result := BatchCreateResult{
		Created: 0,
		Skipped: 0,
		Errors:  []string{},
	}

	if len(input.Transactions) == 0 {
		return result, fmt.Errorf("no transactions provided")
	}

	for _, txInput := range input.Transactions {
		if input.SkipDuplicates {
			tx := transaction.Transaction{
				Date:    timeutil.Date(""),
				Name:    txInput.Name,
				Account: txInput.Account,
			}
			exists, err := s.transactionRepo.Exists(ctx, tx)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("error checking duplicate: %v", err))
				continue
			}
			if exists {
				result.Skipped++
				continue
			}
		}

		_, err := s.CreateTransaction(ctx, txInput)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create transaction %s: %v", txInput.Name, err))
			continue
		}
		result.Created++
	}

	return result, nil
}

func (s *FinanceTransactionsService) UpdateTransaction(ctx context.Context, input UpdateTransactionInput) (FinanceTransactionDTO, error) {
	var errs validation.ValidationErrors

	if err := validation.ValidateRequired("id", input.ID); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateDate("date", input.Date); err != nil {
		errs = append(errs, *err)
	}

	if errs.HasErrors() {
		return FinanceTransactionDTO{}, errs
	}

	var id int
	fmt.Sscanf(input.ID, "%d", &id)

	existing, err := s.transactionRepo.FindByID(ctx, id)
	if err != nil {
		return FinanceTransactionDTO{}, fmt.Errorf("failed to find transaction: %w", err)
	}
	if existing == nil {
		return FinanceTransactionDTO{}, errors.NotFoundError{Resource: "Transaction", ID: input.ID}
	}

	if input.Date != "" {
		txDate, err := timeutil.ParseDate(input.Date)
		if err != nil {
			return FinanceTransactionDTO{}, fmt.Errorf("invalid date format: %w", err)
		}
		existing.Date = txDate
	}
	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.Account != "" {
		existing.Account = input.Account
	}
	if input.Category != "" {
		existing.Category = input.Category
	}
	if input.Note != "" {
		existing.Note = input.Note
	}
	existing.UpdatedAt = time.Now()

	err = s.transactionRepo.Update(ctx, existing)
	if err != nil {
		return FinanceTransactionDTO{}, fmt.Errorf("failed to update transaction: %w", err)
	}

	return FinanceTransactionDTO{
		ID:       fmt.Sprintf("%d", existing.ID),
		Date:     existing.Date.String(),
		Amount:   existing.Amount,
		Account:  existing.Account,
		Category: existing.Category,
		Payee:    existing.Name,
		Notes:    existing.Note,
	}, nil
}

func (s *FinanceTransactionsService) DeleteTransaction(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	var txID int
	fmt.Sscanf(id, "%d", &txID)

	existing, err := s.transactionRepo.FindByID(ctx, txID)
	if err != nil {
		return fmt.Errorf("failed to find transaction: %w", err)
	}
	if existing == nil {
		return errors.NotFoundError{Resource: "Transaction", ID: id}
	}

	err = s.transactionRepo.Delete(ctx, txID)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}
