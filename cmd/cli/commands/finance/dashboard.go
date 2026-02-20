package finance

import (
	"context"
	"encoding/json"
	"fmt"

	"voidline/internal/application/services"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type DashboardCommand struct {
	DBPath string
	Format string
}

func (c *DashboardCommand) Execute(ctx context.Context) error {
	conn, err := sqlite.NewConnection(c.DBPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	accountRepo := sqlite.NewAccountRepository(conn.DB())
	dashboardService := services.NewFinanceDashboardService(accountRepo)

	dashboard, err := dashboardService.GetDashboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get dashboard: %w", err)
	}

	if c.Format == "json" {
		out, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal dashboard: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("\n=== Financial Dashboard ===\n\n")
	fmt.Printf("Total Balance: $%.2f\n", dashboard.TotalBalance)
	fmt.Printf("Net Worth: $%.2f\n", dashboard.NetWorth)
	fmt.Printf("\n--- Accounts (%d) ---\n", len(dashboard.Accounts))
	for _, acc := range dashboard.Accounts {
		fmt.Printf("  %s (%s): $%.2f\n", acc.Name, acc.Type, acc.Balance)
	}
	fmt.Println()
	return nil
}
