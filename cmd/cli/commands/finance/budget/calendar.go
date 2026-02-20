package budget

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	calendarHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	incomeStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	expenseStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	balanceStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	weekendStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	todayStyle          = lipgloss.NewStyle().Background(lipgloss.Color("86")).Foreground(lipgloss.Color("0"))
)

// CalendarCommand displays a monthly cash flow calendar
func CalendarCommand(month string, showBalances bool) int {

	// Load budget
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	// Determine month
	displayMonth := time.Now()
	if month != "" {
		parsed, err := time.Parse("2006-01", month)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Invalid month format. Use YYYY-MM\n")
			return 1
		}
		displayMonth = parsed
	}

	// Generate calendar
	return generateCalendar(config, displayMonth, showBalances)
}

func generateCalendar(config *BudgetConfig, month time.Time, showBalances bool) int {
	fmt.Println()
	fmt.Println(calendarHeaderStyle.Render(fmt.Sprintf("📅 Cash Flow Calendar - %s", month.Format("January 2006"))))
	fmt.Println()

	// Calculate starting balance
	startingBalance := config.Settings.SafetyBuffer

	// Collect all transactions for this month
	type Transaction struct {
		Day    int
		Name   string
		Amount float64
		Type   string // "income" or "expense"
	}

	var transactions []Transaction

	// Add income
	for _, income := range config.CashFlow.Income {
		// Check if this income occurs this month
		if income.Month == 0 || income.Month == int(month.Month()) {
			transactions = append(transactions, Transaction{
				Day:    income.DayOfMonth,
				Name:   income.Name,
				Amount: income.Amount,
				Type:   "income",
			})
		}
	}

	// Add expenses
	for _, expense := range config.CashFlow.Expenses {
		transactions = append(transactions, Transaction{
			Day:    expense.DayOfMonth,
			Name:   expense.Name,
			Amount: expense.Amount,
			Type:   "expense",
		})
	}

	// Sort by day
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Day < transactions[j].Day
	})

	// Print day-by-day view
	fmt.Println(calendarHeaderStyle.Render("Daily Cash Flow:"))
	fmt.Println()

	currentBalance := startingBalance
	lastDay := 0

	for _, tx := range transactions {
		// Show days with no activity
		if tx.Day > lastDay+1 && lastDay > 0 {
			for d := lastDay + 1; d < tx.Day; d++ {
				fmt.Printf("  Day %2d: No activity", d)
				if showBalances {
					fmt.Printf("  (Balance: %s)", FormatCurrency(currentBalance))
				}
				fmt.Println()
			}
		}

		// Show transaction
		symbol := "💸"
		style := expenseStyle
		if tx.Type == "income" {
			symbol = "💰"
			style = incomeStyle
			currentBalance += tx.Amount
		} else {
			currentBalance -= tx.Amount
		}

		fmt.Printf("  Day %2d: %s %s ", tx.Day, symbol, style.Render(FormatCurrency(tx.Amount)))
		fmt.Printf("%-30s", tx.Name)
		if showBalances {
			fmt.Printf("  Balance: %s", balanceStyle.Render(FormatCurrency(currentBalance)))
		}
		fmt.Println()

		lastDay = tx.Day
	}

	// Show remaining days
	daysInMonth := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if lastDay < daysInMonth {
		fmt.Println()
		fmt.Printf("  Days %d-%d: No activity", lastDay+1, daysInMonth)
		if showBalances {
			fmt.Printf("  (Balance: %s)", FormatCurrency(currentBalance))
		}
		fmt.Println()
	}

	fmt.Println()

	// Summary
	monthlyIncome := config.CashFlow.CalculateMonthlyIncome()
	monthlyExpenses := config.CashFlow.CalculateMonthlyExpenses()
	surplus := monthlyIncome - monthlyExpenses

	fmt.Println(calendarHeaderStyle.Render("Month Summary:"))
	fmt.Printf("  Starting Balance: %s\n", FormatCurrency(startingBalance))
	fmt.Printf("  Total Income:     %s\n", incomeStyle.Render(FormatCurrency(monthlyIncome)))
	fmt.Printf("  Total Expenses:   %s\n", expenseStyle.Render(FormatCurrency(monthlyExpenses)))

	if surplus >= 0 {
		fmt.Printf("  Surplus:          %s\n", incomeStyle.Render(FormatCurrency(surplus)))
	} else {
		fmt.Printf("  Deficit:          %s\n", expenseStyle.Render(FormatCurrency(surplus)))
	}

	fmt.Printf("  Ending Balance:   %s\n", balanceStyle.Render(FormatCurrency(currentBalance)))
	fmt.Println()

	// Critical dates
	fmt.Println(calendarHeaderStyle.Render("Critical Dates:"))
	lowBalanceDay := -1
	lowestBalance := currentBalance
	tempBalance := startingBalance

	for day := 1; day <= daysInMonth; day++ {
		// Apply transactions for this day
		for _, tx := range transactions {
			if tx.Day == day {
				if tx.Type == "income" {
					tempBalance += tx.Amount
				} else {
					tempBalance -= tx.Amount
				}
			}
		}

		if tempBalance < lowestBalance {
			lowestBalance = tempBalance
			lowBalanceDay = day
		}
	}

	if lowBalanceDay > 0 {
		fmt.Printf("  ⚠️  Lowest balance on Day %d: %s\n", lowBalanceDay, FormatCurrency(lowestBalance))
	}

	// Find largest single expense
	var largestExpense Transaction
	for _, tx := range transactions {
		if tx.Type == "expense" && tx.Amount > largestExpense.Amount {
			largestExpense = tx
		}
	}

	if largestExpense.Amount > 0 {
		fmt.Printf("  💸 Largest expense on Day %d: %s (%s)\n",
			largestExpense.Day,
			FormatCurrency(largestExpense.Amount),
			largestExpense.Name)
	}

	// Find largest income
	var largestIncome Transaction
	for _, tx := range transactions {
		if tx.Type == "income" && tx.Amount > largestIncome.Amount {
			largestIncome = tx
		}
	}

	if largestIncome.Amount > 0 {
		fmt.Printf("  💰 Largest income on Day %d: %s (%s)\n",
			largestIncome.Day,
			FormatCurrency(largestIncome.Amount),
			largestIncome.Name)
	}

	fmt.Println()
	return 0
}
