package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"

	"voidline/internal/application/services"
	"voidline/internal/infrastructure/persistence/sqlite"
)

var reportTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC"))

// ReportCommand handles report generation (moved under finance)
type ReportCommand struct {
	DBPath      string
	ReportType  string
	Format      string
	Output      string
	Page        int
	PerPage     int
	StartDate   string
	EndDate     string
	Accounts    []string
	Categories  []string
	SearchQuery string
}

// Execute runs the report command
func (c *ReportCommand) Execute(ctx context.Context) error {
	// Connect to database
	conn, err := sqlite.NewConnection(c.DBPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	txRepo := sqlite.NewTransactionRepository(conn.DB())
	accountRepo := sqlite.NewAccountRepository(conn.DB())
	categoryRepo := sqlite.NewCategoryRepository(conn.DB())

	txService := services.NewFinanceTransactionsService(txRepo)
	accountService := services.NewFinanceAccountsService(accountRepo, txRepo)
	categoryService := services.NewFinanceCategoriesService(categoryRepo)

	// Handle different report types
	switch strings.ToLower(c.ReportType) {
	case "transactions":
		return c.executeTransactions(ctx, txService)
	case "accounts":
		return c.executeAccounts(ctx, accountService)
	case "categories":
		return c.executeCategories(ctx, categoryService)
	default:
		return fmt.Errorf("unknown report type: %s", c.ReportType)
	}
}

func (c *ReportCommand) executeTransactions(ctx context.Context, txService *services.FinanceTransactionsService) error {
	opts := services.GetTransactionsOptions{
		Page:      c.Page,
		PerPage:   c.PerPage,
		StartDate: c.StartDate,
		EndDate:   c.EndDate,
	}
	if len(c.Accounts) > 0 {
		opts.Account = c.Accounts[0]
	}
	if len(c.Categories) > 0 {
		opts.Category = c.Categories[0]
	}

	txs, err := txService.GetTransactions(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to get transactions: %w", err)
	}

	switch strings.ToLower(c.Format) {
	case "json":
		return c.outputJSON(txs)
	case "table":
		return c.outputTransactionsTable(txs)
	case "tui":
		return runTransactionsTUI(txs)
	default:
		return fmt.Errorf("unknown output format: %s", c.Format)
	}
}

func (c *ReportCommand) executeAccounts(ctx context.Context, accountService *services.FinanceAccountsService) error {
	accounts, err := accountService.GetAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	switch strings.ToLower(c.Format) {
	case "json":
		return c.outputJSON(accounts)
	case "table":
		return c.outputAccountsTable(accounts)
	case "tui":
		return runAccountsTUI(accounts)
	default:
		return fmt.Errorf("unknown output format: %s", c.Format)
	}
}

func (c *ReportCommand) executeCategories(ctx context.Context, categoryService *services.FinanceCategoriesService) error {
	categories, err := categoryService.GetCategories(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get categories: %w", err)
	}

	switch strings.ToLower(c.Format) {
	case "json":
		return c.outputJSON(categories)
	case "table":
		return c.outputCategoriesTable(categories)
	case "tui":
		return runCategoriesTUI(categories)
	default:
		return fmt.Errorf("unknown output format: %s", c.Format)
	}
}

func (c *ReportCommand) outputJSON(data interface{}) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if c.Output == "" {
		fmt.Println(string(out))
	} else {
		if err := os.WriteFile(c.Output, out, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	}
	return nil
}

func (c *ReportCommand) outputTransactionsTable(txs []services.FinanceTransactionDTO) error {
	// Styled title using charm's lipgloss
	fmt.Println()
	fmt.Println(reportTitleStyle.Render("Transactions"))
	if len(txs) > 0 {
		fmt.Printf("Page %d\n\n", c.Page)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join([]string{"Date", "Account", "Name", "Category", "Amount"}, "\t"))

	separators := []string{"----", "------------------------------", "----------------------------------------", "--------------------", "------------"}
	fmt.Fprintln(w, strings.Join(separators, "\t"))

	for _, tx := range txs {
		amountStr := fmt.Sprintf("%.2f", tx.Amount)
		fmt.Fprintln(w, strings.Join([]string{tx.Date, tx.Account, tx.Payee, tx.Category, amountStr}, "\t"))
	}

	w.Flush()
	fmt.Printf("\nTotal: %d\n\n", len(txs))
	return nil
}

func (c *ReportCommand) outputAccountsTable(accounts []services.FinanceAccountDTO) error {
	fmt.Println()
	fmt.Println(reportTitleStyle.Render("Accounts"))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join([]string{"Name", "Type", "Balance", "Currency", "Active"}, "\t"))

	separators := []string{"------------------------------", "------------", "---------------", "---------", "-------"}
	fmt.Fprintln(w, strings.Join(separators, "\t"))

	for _, acc := range accounts {
		active := "Yes"
		if !acc.IsActive {
			active = "No"
		}
		fmt.Fprintln(w, strings.Join([]string{acc.Name, acc.Type, fmt.Sprintf("%.2f", acc.Balance), acc.Currency, active}, "\t"))
	}

	w.Flush()
	fmt.Println()
	return nil
}

func (c *ReportCommand) outputCategoriesTable(categories []services.FinanceCategoryDTO) error {
	fmt.Println()
	fmt.Println(reportTitleStyle.Render("Categories"))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join([]string{"Name"}, "\t"))

	separators := []string{"--------------------"}
	fmt.Fprintln(w, strings.Join(separators, "\t"))

	for _, cat := range categories {
		fmt.Fprintln(w, cat.Name)
	}

	w.Flush()
	fmt.Println()
	return nil
}
