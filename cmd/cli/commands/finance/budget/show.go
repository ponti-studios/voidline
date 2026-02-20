package budget

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	headerStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color("86"))
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	labelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	positiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	negativeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)

// ShowCommand displays the current budget status
func ShowCommand(view, month string) int {

	// Load budget
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	// Determine which month to show
	showMonth := time.Now()
	if month != "" {
		parsed, err := time.Parse("2006-01", month)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Invalid month format. Use YYYY-MM\n")
			return 1
		}
		showMonth = parsed
	}

	// Route to appropriate view
	switch view {
	case "categories":
		return showCategoriesView(config)
	case "cashflow":
		return showCashFlowView(config, showMonth)
	case "goals":
		return showGoalsView(config)
	default:
		return showSummaryView(config, showMonth)
	}
}

// showSummaryView displays the main budget summary
func showSummaryView(config *BudgetConfig, month time.Time) int {
	// Calculate totals
	monthlyIncome := config.CashFlow.CalculateMonthlyIncome()
	monthlyExpenses := config.CashFlow.CalculateMonthlyExpenses()
	essentialExpenses := config.CashFlow.CalculateEssentialExpenses()
	discretionaryExpenses := config.CashFlow.CalculateDiscretionaryExpenses()
	surplus := monthlyIncome - monthlyExpenses

	// Header
	monthName := month.Format("January 2006")
	headerContent := fmt.Sprintf("%s\n%s\n", config.Name, monthName)
	headerContent += fmt.Sprintf("Income: %s | Expenses: %s\n",
		FormatCurrency(monthlyIncome), FormatCurrency(monthlyExpenses))

	if surplus >= 0 {
		headerContent += fmt.Sprintf("Surplus: %s", positiveStyle.Render(FormatCurrency(surplus)))
	} else {
		headerContent += fmt.Sprintf("Deficit: %s", negativeStyle.Render(FormatCurrency(surplus)))
	}

	fmt.Println(headerStyle.Render(headerContent))
	fmt.Println()

	// Income section
	fmt.Println(titleStyle.Render("💰 Income Sources"))
	if len(config.CashFlow.Income) == 0 {
		fmt.Println("  No income sources configured")
	} else {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Source", "Amount", "Day"})

		for _, income := range config.CashFlow.Income {
			day := fmt.Sprintf("%d", income.DayOfMonth)
			if income.Month > 0 {
				day = fmt.Sprintf("%s %d", time.Month(income.Month).String()[:3], income.DayOfMonth)
			}
			t.AppendRow([]interface{}{income.Name, FormatCurrency(income.Amount), day})
		}

		t.AppendFooter(table.Row{"Total", FormatCurrency(monthlyIncome), ""})
		t.Render()
	}
	fmt.Println()

	// Expenses section
	fmt.Println(titleStyle.Render("💸 Expenses"))
	if len(config.CashFlow.Expenses) == 0 {
		fmt.Println("  No expenses configured")
	} else {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Category", "Amount", "Type", "Essential"})

		for _, expense := range config.CashFlow.Expenses {
			typeStr := expense.Type
			essentialStr := "❌"
			if expense.Essential {
				essentialStr = "✅"
			}

			// Color-code the amount
			amountStr := FormatCurrency(expense.Amount)
			if expense.Essential {
				amountStr = labelStyle.Render(amountStr)
			}

			t.AppendRow([]interface{}{expense.Name, amountStr, typeStr, essentialStr})
		}

		t.AppendFooter(table.Row{"Total", FormatCurrency(monthlyExpenses), "", ""})
		t.Render()
	}
	fmt.Println()

	// Breakdown
	fmt.Println(titleStyle.Render("📊 Breakdown"))
	fmt.Printf("  %s Essential:      %s (%.1f%%)\n",
		labelStyle.Render("•"),
		FormatCurrency(essentialExpenses),
		(essentialExpenses/monthlyExpenses)*100)
	fmt.Printf("  %s Discretionary: %s (%.1f%%)\n",
		labelStyle.Render("•"),
		FormatCurrency(discretionaryExpenses),
		(discretionaryExpenses/monthlyExpenses)*100)
	fmt.Println()

	// Goals progress
	if len(config.Goals) > 0 {
		fmt.Println(titleStyle.Render("🎯 Goals Progress"))

		for _, goal := range config.Goals {
			progress := 0.0
			if goal.Target > 0 {
				progress = (goal.Current / goal.Target) * 100
			}

			// Progress bar
			barWidth := 20
			filled := int((progress / 100) * float64(barWidth))
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			status := ""
			if progress >= 100 {
				status = positiveStyle.Render(" ✅ Complete!")
			} else if surplus > 0 && goal.TimelineMonths > 0 {
				requiredMonthly := (goal.Target - goal.Current) / float64(goal.TimelineMonths)
				if requiredMonthly <= surplus {
					monthsLeft := int((goal.Target - goal.Current) / surplus)
					status = fmt.Sprintf(" (%d months to go)", monthsLeft)
				} else {
					status = warningStyle.Render(" ⚠️  Need more savings")
				}
			}

			fmt.Printf("  %s\n", goal.Name)
			fmt.Printf("  %s %.0f%% (%s / %s)%s\n",
				progressStyle.Render(bar),
				progress,
				FormatCurrency(goal.Current),
				FormatCurrency(goal.Target),
				status)
			fmt.Println()
		}
	}

	// Insights
	fmt.Println(titleStyle.Render("💡 Insights"))

	if surplus > 0 {
		savingsRate := (surplus / monthlyIncome) * 100
		fmt.Printf("  ✅ Savings rate: %.1f%% (recommended: 20%%+)\n", savingsRate)

		if savingsRate >= 20 {
			fmt.Printf("  %s\n", positiveStyle.Render("  Great job! You're saving more than 20% of your income."))
		} else if savingsRate >= 10 {
			fmt.Printf("  %s\n", warningStyle.Render("  Good start. Try to increase to 20% for better financial security."))
		} else {
			fmt.Printf("  %s\n", negativeStyle.Render("  Consider reducing discretionary spending to increase savings."))
		}

		// Runway calculation (assuming current expenses)
		if essentialExpenses > 0 {
			runway := surplus / essentialExpenses
			fmt.Printf("  • Runway on essential expenses only: %.1f months\n", runway)
		}
	} else {
		fmt.Printf("  %s\n", negativeStyle.Render("  ⚠️  Your expenses exceed income. Consider:"))
		fmt.Println("    - Reviewing discretionary spending (can be reduced)")
		fmt.Println("    - Finding additional income sources")
		fmt.Println("    - Extending goal timelines")

		if discretionaryExpenses > 0 {
			reductionNeeded := -surplus
			fmt.Printf("  \n  To break even, reduce discretionary spending by %s\n",
				FormatCurrency(reductionNeeded))
		}
	}

	// 50/30/20 rule check
	if monthlyIncome > 0 {
		needsPct := (essentialExpenses / monthlyIncome) * 100
		wantsPct := (discretionaryExpenses / monthlyIncome) * 100
		savingsPct := (surplus / monthlyIncome) * 100

		fmt.Printf("\n  50/30/20 Rule Analysis:\n")
		fmt.Printf("    Needs (50%%):   %s (%.1f%%) %s\n",
			FormatCurrency(essentialExpenses), needsPct,
			checkRule(needsPct, 50))
		fmt.Printf("    Wants (30%%):   %s (%.1f%%) %s\n",
			FormatCurrency(discretionaryExpenses), wantsPct,
			checkRule(wantsPct, 30))
		fmt.Printf("    Savings (20%%): %s (%.1f%%) %s\n",
			FormatCurrency(surplus), savingsPct,
			checkRule(savingsPct, 20))
	}

	fmt.Println()

	return 0
}

// checkRule returns emoji indicator for 50/30/20 rule
func checkRule(actual, target float64) string {
	diff := actual - target
	if diff >= -5 && diff <= 5 {
		return positiveStyle.Render("✅")
	} else if diff > 5 {
		return warningStyle.Render("⚠️  high")
	}
	return ""
}

// showCategoriesView displays expenses grouped by category
func showCategoriesView(config *BudgetConfig) int {
	totals := config.CashFlow.GetTotalByCategory()
	monthlyExpenses := config.CashFlow.CalculateMonthlyExpenses()

	fmt.Println(headerStyle.Render("Expenses by Category"))
	fmt.Println()

	if len(totals) == 0 {
		fmt.Println("No expenses configured.")
		return 0
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Category", "Amount", "% of Total", "Bar"})

	// Sort categories by amount (would need to implement sorting)
	for category, amount := range totals {
		percentage := (amount / monthlyExpenses) * 100

		// Create bar
		barWidth := 20
		filled := int((percentage / 100) * float64(barWidth))
		if filled < 1 && amount > 0 {
			filled = 1
		}
		bar := strings.Repeat("█", filled)

		t.AppendRow([]interface{}{
			category,
			FormatCurrency(amount),
			fmt.Sprintf("%.1f%%", percentage),
			bar,
		})
	}

	t.AppendFooter(table.Row{"Total", FormatCurrency(monthlyExpenses), "100%", ""})
	t.Render()

	return 0
}

// showCashFlowView displays daily cash flow for a month
func showCashFlowView(config *BudgetConfig, month time.Time) int {
	fmt.Println(headerStyle.Render(fmt.Sprintf("Cash Flow Calendar - %s", month.Format("January 2006"))))
	fmt.Println()

	// This would show a calendar view with income/expenses on specific days
	// For now, show a list view
	fmt.Println(titleStyle.Render("Income"))
	for _, income := range config.CashFlow.Income {
		if income.Month == 0 || income.Month == int(month.Month()) {
			fmt.Printf("  Day %2d: %s %s\n",
				income.DayOfMonth,
				positiveStyle.Render("+"+FormatCurrency(income.Amount)),
				income.Name)
		}
	}

	fmt.Println()
	fmt.Println(titleStyle.Render("Expenses"))
	for _, expense := range config.CashFlow.Expenses {
		fmt.Printf("  Day %2d: %s %s",
			expense.DayOfMonth,
			negativeStyle.Render("-"+FormatCurrency(expense.Amount)),
			expense.Name)
		if expense.Essential {
			fmt.Print(" (essential)")
		}
		fmt.Println()
	}

	return 0
}

// showGoalsView displays detailed goal information
func showGoalsView(config *BudgetConfig) int {
	fmt.Println(headerStyle.Render("Financial Goals"))
	fmt.Println()

	if len(config.Goals) == 0 {
		fmt.Println("No goals configured. Run 'hominem budget init' to set goals.")
		return 0
	}

	monthlyIncome := config.CashFlow.CalculateMonthlyIncome()
	monthlyExpenses := config.CashFlow.CalculateMonthlyExpenses()
	surplus := monthlyIncome - monthlyExpenses

	for i, goal := range config.Goals {
		progress := 0.0
		if goal.Target > 0 {
			progress = (goal.Current / goal.Target) * 100
		}

		remaining := goal.Target - goal.Current

		fmt.Printf("%s\n", titleStyle.Render(fmt.Sprintf("%d. %s", i+1, goal.Name)))
		fmt.Printf("   Target: %s\n", FormatCurrency(goal.Target))
		fmt.Printf("   Current: %s (%.1f%%)\n", FormatCurrency(goal.Current), progress)
		fmt.Printf("   Remaining: %s\n", FormatCurrency(remaining))
		fmt.Printf("   Timeline: %d months\n", goal.TimelineMonths)
		fmt.Printf("   Priority: %d\n", goal.Priority)

		if surplus > 0 && remaining > 0 {
			monthsToGoal := int(remaining / surplus)
			fmt.Printf("   At current savings rate: %d months to complete\n", monthsToGoal)

			if monthsToGoal <= goal.TimelineMonths {
				fmt.Printf("   %s\n", positiveStyle.Render("   ✅ On track to meet goal!"))
			} else {
				fmt.Printf("   %s\n", warningStyle.Render("   ⚠️  Need to save more or extend timeline"))

				requiredMonthly := remaining / float64(goal.TimelineMonths)
				fmt.Printf("   Required monthly savings: %s (currently: %s)\n",
					FormatCurrency(requiredMonthly),
					FormatCurrency(surplus))
			}
		}

		fmt.Println()
	}

	return 0
}
