package finance

import (
	"fmt"

	"github.com/spf13/cobra"

	"gogogo/cmd/cli/commands/finance/budget"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finance",
		Short: "Finance and budget utilities",
	}

	cmd.AddCommand(budgetCmd())
	cmd.AddCommand(calculatorCmd())
	cmd.AddCommand(reportCmd())
	cmd.AddCommand(dashboardCmd())

	return cmd
}

func budgetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "Budget management",
	}

	cmd.AddCommand(budgetInitCmd())
	cmd.AddCommand(budgetShowCmd())
	cmd.AddCommand(budgetCalendarCmd())
	cmd.AddCommand(budgetScenarioCmd())
	cmd.AddCommand(budgetExportCmd())

	return cmd
}

func budgetInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a new budget interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			if code := budget.InitCommand(); code != 0 {
				return fmt.Errorf("budget init failed with exit code %d", code)
			}
			return nil
		},
	}
}

func budgetShowCmd() *cobra.Command {
	var view string
	var month string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current budget status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if code := budget.ShowCommand(view, month); code != 0 {
				return fmt.Errorf("budget show failed with exit code %d", code)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&view, "view", "summary", "View type: summary, categories, cashflow, goals")
	cmd.Flags().StringVar(&month, "month", "", "Month to show (YYYY-MM format, default: current)")

	return cmd
}

func budgetCalendarCmd() *cobra.Command {
	var month string
	var showBalances bool

	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "Show cash flow calendar",
		RunE: func(cmd *cobra.Command, args []string) error {
			if code := budget.CalendarCommand(month, showBalances); code != 0 {
				return fmt.Errorf("budget calendar failed with exit code %d", code)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&month, "month", "", "Month to display (YYYY-MM, default: current)")
	cmd.Flags().BoolVar(&showBalances, "balances", true, "Show running balances")

	return cmd
}

func budgetScenarioCmd() *cobra.Command {
	var opts budget.ScenarioOptions

	cmd := &cobra.Command{
		Use:   "scenario",
		Short: "Test what-if scenarios",
		RunE: func(cmd *cobra.Command, args []string) error {
			if code := budget.ScenarioCommand(opts); code != 0 {
				return fmt.Errorf("budget scenario failed with exit code %d", code)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Scenario name (optional)")
	cmd.Flags().StringVar(&opts.ReduceExpense, "reduce-expense", "", "Reduce expense by percentage (format: 'Expense Name:50')")
	cmd.Flags().StringVar(&opts.IncreaseExpense, "increase-expense", "", "Increase expense by percentage (format: 'Expense Name:50')")
	cmd.Flags().StringVar(&opts.RemoveExpense, "remove-expense", "", "Remove expense entirely")
	cmd.Flags().StringVar(&opts.AddExpense, "add-expense", "", "Add new expense (format: 'Name:Amount')")
	cmd.Flags().StringVar(&opts.ReduceIncome, "reduce-income", "", "Reduce income by percentage (format: 'Income Name:50')")
	cmd.Flags().StringVar(&opts.IncreaseIncome, "increase-income", "", "Increase income by percentage (format: 'Income Name:50')")
	cmd.Flags().StringVar(&opts.AddIncome, "add-income", "", "Add new income (format: 'Name:Amount')")
	cmd.Flags().StringVar(&opts.ExtendGoal, "extend-goal", "", "Extend goal timeline (format: 'Goal Name:6' months)")

	return cmd
}

func budgetExportCmd() *cobra.Command {
	var format string
	var output string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export budget to various formats",
		RunE: func(cmd *cobra.Command, args []string) error {
			if code := budget.ExportCommand(format, output); code != 0 {
				return fmt.Errorf("budget export failed with exit code %d", code)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "csv", "Export format: csv, json, yaml")
	cmd.Flags().StringVar(&output, "output", "", "Output file (default: stdout)")

	return cmd
}

func calculatorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "calculator",
		Short: "Goal calculator (interactive TUI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunCLI()
		},
	}
}

func reportCmd() *cobra.Command {
	var dbPath string
	var reportType string
	var format string
	var output string
	var page int
	var perPage int
	var startDate string
	var endDate string
	var accounts []string
	var categories []string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate financial reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			reportCmd := ReportCommand{
				DBPath:     dbPath,
				ReportType: reportType,
				Format:     format,
				Output:     output,
				Page:       page,
				PerPage:    perPage,
				StartDate:  startDate,
				EndDate:    endDate,
				Accounts:   accounts,
				Categories: categories,
			}
			return reportCmd.Execute(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (required)")
	cmd.Flags().StringVar(&reportType, "type", "", "Report type: transactions, accounts, categories (required)")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json, tui")
	cmd.Flags().StringVar(&output, "output", "", "Output file (optional, prints to stdout if not specified)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&perPage, "per-page", 50, "Items per page")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringArrayVar(&accounts, "account", []string{}, "Filter by account")
	cmd.Flags().StringArrayVar(&categories, "category", []string{}, "Filter by category")

	if err := cmd.MarkFlagRequired("db"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("type"); err != nil {
		panic(err)
	}

	return cmd
}

func dashboardCmd() *cobra.Command {
	var dbPath string
	var format string

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Show financial dashboard summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			dashboardCmd := DashboardCommand{
				DBPath: dbPath,
				Format: format,
			}
			return dashboardCmd.Execute(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (required)")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")

	if err := cmd.MarkFlagRequired("db"); err != nil {
		panic(err)
	}

	return cmd
}
