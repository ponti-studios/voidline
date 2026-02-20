package services

import (
	"context"
	"fmt"
	"time"

	"voidline/internal/application/errors"
	"voidline/internal/application/validation"
	"voidline/internal/domain/account"
	"voidline/internal/domain/transaction"
)

type FinanceAccountsService struct {
	accountRepo     account.Repository
	transactionRepo transaction.Repository
}

func NewFinanceAccountsService(accountRepo account.Repository, transactionRepo transaction.Repository) *FinanceAccountsService {
	return &FinanceAccountsService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

type FinanceAccountDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Balance     float64 `json:"balance"`
	Currency    string  `json:"currency"`
	IsActive    bool    `json:"isActive"`
	LastUpdated string  `json:"lastUpdated"`
}

type CreateAccountInput struct {
	Name     string `json:"name" validate:"required"`
	Type     string `json:"type" validate:"required"`
	Currency string `json:"currency" validate:"required"`
}

type UpdateAccountInput struct {
	ID       string `json:"id" validate:"required"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsActive *bool  `json:"isActive"`
}

func (s *FinanceAccountsService) GetAccounts(ctx context.Context) ([]FinanceAccountDTO, error) {
	accounts, err := s.accountRepo.FindActive(ctx)
	if err != nil {
		return nil, err
	}

	balances, err := s.accountRepo.GetBalances(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]FinanceAccountDTO, len(accounts))
	for i, acc := range accounts {
		dtos[i] = FinanceAccountDTO{
			ID:          fmt.Sprintf("%d", acc.ID),
			Name:        acc.Name,
			Type:        string(acc.Type),
			Balance:     balances[acc.Name],
			Currency:    acc.Currency,
			IsActive:    acc.IsActive,
			LastUpdated: "",
		}
	}

	return dtos, nil
}

func (s *FinanceAccountsService) CreateAccount(ctx context.Context, input CreateAccountInput) (FinanceAccountDTO, error) {
	var errs validation.ValidationErrors

	if err := validation.ValidateRequired("name", input.Name); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateRequired("type", input.Type); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateRequired("currency", input.Currency); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateAccountType("type", input.Type); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateCurrency("currency", input.Currency); err != nil {
		errs = append(errs, *err)
	}

	if errs.HasErrors() {
		return FinanceAccountDTO{}, errs
	}

	accountType := account.Type(input.Type)

	acc := &account.Account{
		Name:     input.Name,
		Type:     accountType,
		Currency: input.Currency,
		IsActive: true,
	}

	err := s.accountRepo.Create(ctx, acc)
	if err != nil {
		return FinanceAccountDTO{}, fmt.Errorf("failed to create account: %w", err)
	}

	return FinanceAccountDTO{
		ID:          fmt.Sprintf("%d", acc.ID),
		Name:        acc.Name,
		Type:        string(acc.Type),
		Balance:     0,
		Currency:    acc.Currency,
		IsActive:    acc.IsActive,
		LastUpdated: time.Now().Format("2006-01-02"),
	}, nil
}

func (s *FinanceAccountsService) UpdateAccount(ctx context.Context, input UpdateAccountInput) (FinanceAccountDTO, error) {
	var errs validation.ValidationErrors

	if err := validation.ValidateRequired("id", input.ID); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateAccountType("type", input.Type); err != nil {
		errs = append(errs, *err)
	}

	if errs.HasErrors() {
		return FinanceAccountDTO{}, errs
	}

	var id int
	fmt.Sscanf(input.ID, "%d", &id)

	existing, err := s.accountRepo.FindByID(ctx, id)
	if err != nil {
		return FinanceAccountDTO{}, fmt.Errorf("failed to find account: %w", err)
	}
	if existing == nil {
		return FinanceAccountDTO{}, errors.NotFoundError{Resource: "Account", ID: input.ID}
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.Type != "" {
		existing.Type = account.Type(input.Type)
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}

	err = s.accountRepo.Update(ctx, existing)
	if err != nil {
		return FinanceAccountDTO{}, fmt.Errorf("failed to update account: %w", err)
	}

	balances, err := s.accountRepo.GetBalances(ctx)
	if err != nil {
		return FinanceAccountDTO{}, fmt.Errorf("failed to get balance: %w", err)
	}

	return FinanceAccountDTO{
		ID:          fmt.Sprintf("%d", existing.ID),
		Name:        existing.Name,
		Type:        string(existing.Type),
		Balance:     balances[existing.Name],
		Currency:    existing.Currency,
		IsActive:    existing.IsActive,
		LastUpdated: time.Now().Format("2006-01-02"),
	}, nil
}

func (s *FinanceAccountsService) DeleteAccount(ctx context.Context, id string) (int, error) {
	if err := validation.ValidateRequired("id", id); err != nil {
		return 0, err
	}

	var accID int
	fmt.Sscanf(id, "%d", &accID)

	existing, err := s.accountRepo.FindByID(ctx, accID)
	if err != nil {
		return 0, fmt.Errorf("failed to find account: %w", err)
	}
	if existing == nil {
		return 0, errors.NotFoundError{Resource: "Account", ID: id}
	}

	filter := transaction.NewFilter()
	filter.Accounts = []string{existing.Name}

	txs, err := s.transactionRepo.FindByFilter(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to find transactions: %w", err)
	}

	deletedCount := 0
	for _, tx := range txs.Items {
		err = s.transactionRepo.Delete(ctx, tx.ID)
		if err != nil {
			return deletedCount, errors.CascadeDeleteError{Count: len(txs.Items), Message: fmt.Sprintf("failed to delete transaction %d", tx.ID)}
		}
		deletedCount++
	}

	err = s.accountRepo.Delete(ctx, accID)
	if err != nil {
		return deletedCount, fmt.Errorf("failed to delete account: %w", err)
	}

	return deletedCount, nil
}
