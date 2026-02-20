package services

import (
	"context"
	"fmt"

	"voidline/internal/domain/account"
)

type FinanceDashboardService struct {
	accountRepo account.Repository
}

func NewFinanceDashboardService(accountRepo account.Repository) *FinanceDashboardService {
	return &FinanceDashboardService{accountRepo: accountRepo}
}

type FinanceDashboardDTO struct {
	TotalBalance float64             `json:"totalBalance"`
	NetWorth     float64             `json:"netWorth"`
	Accounts     []FinanceAccountDTO `json:"accounts"`
}

func (s *FinanceDashboardService) GetDashboard(ctx context.Context) (FinanceDashboardDTO, error) {
	accounts, err := s.accountRepo.FindActive(ctx)
	if err != nil {
		return FinanceDashboardDTO{}, err
	}

	balances, err := s.accountRepo.GetBalances(ctx)
	if err != nil {
		return FinanceDashboardDTO{}, err
	}

	var totalAssets, totalLiabilities float64
	accountDTOs := make([]FinanceAccountDTO, len(accounts))

	for i, acc := range accounts {
		balance := balances[acc.Name]
		acc.CurrentBalance = balance
		displayBalance := acc.DisplayBalance()

		if acc.Type.IsAsset() {
			totalAssets += displayBalance
		} else {
			totalLiabilities += displayBalance
		}

		accountDTOs[i] = FinanceAccountDTO{
			ID:       fmt.Sprintf("%d", acc.ID),
			Name:     acc.Name,
			Type:     string(acc.Type),
			Balance:  displayBalance,
			Currency: acc.Currency,
			IsActive: acc.IsActive,
		}
	}

	netWorth := totalAssets - totalLiabilities

	return FinanceDashboardDTO{
		TotalBalance: totalAssets,
		NetWorth:     netWorth,
		Accounts:     accountDTOs,
	}, nil
}
