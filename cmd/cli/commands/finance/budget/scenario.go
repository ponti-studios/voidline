package budget

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

var (
	scenarioTitleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	positiveImpactStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	negativeImpactStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	neutralImpactStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	recommendationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
)

type ScenarioOptions struct {
	Name            string
	ReduceExpense   string
	IncreaseExpense string
	RemoveExpense   string
	AddExpense      string
	ReduceIncome    string
	IncreaseIncome  string
	AddIncome       string
	ExtendGoal      string
}

// ScenarioCommand allows testing what-if scenarios
func ScenarioCommand(opts ScenarioOptions) int {

	// Load budget
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	// Create scenario name if not provided
	scenarioName := opts.Name
	if scenarioName == "" {
		scenarioName = fmt.Sprintf("Scenario_%s", time.Now().Format("20060102_150405"))
	}

	// Clone config for scenario
	scenario := cloneConfig(config)
	changes := []string{}

	// Apply expense reductions
	if opts.ReduceExpense != "" {
		parts := strings.Split(opts.ReduceExpense, ":")
		if len(parts) == 2 {
			expenseName := parts[0]
			var percentage float64
			fmt.Sscanf(parts[1], "%f", &percentage)

			for i := range scenario.CashFlow.Expenses {
				if strings.EqualFold(scenario.CashFlow.Expenses[i].Name, expenseName) {
					oldAmount := scenario.CashFlow.Expenses[i].Amount
					scenario.CashFlow.Expenses[i].Amount = oldAmount * (1 - percentage/100)
					changes = append(changes, fmt.Sprintf("Reduced %s by %.0f%% (%s → %s)",
						expenseName, percentage, FormatCurrency(oldAmount),
						FormatCurrency(scenario.CashFlow.Expenses[i].Amount)))
					break
				}
			}
		}
	}

	// Apply expense increases
	if opts.IncreaseExpense != "" {
		parts := strings.Split(opts.IncreaseExpense, ":")
		if len(parts) == 2 {
			expenseName := parts[0]
			var percentage float64
			fmt.Sscanf(parts[1], "%f", &percentage)

			for i := range scenario.CashFlow.Expenses {
				if strings.EqualFold(scenario.CashFlow.Expenses[i].Name, expenseName) {
					oldAmount := scenario.CashFlow.Expenses[i].Amount
					scenario.CashFlow.Expenses[i].Amount = oldAmount * (1 + percentage/100)
					changes = append(changes, fmt.Sprintf("Increased %s by %.0f%% (%s → %s)",
						expenseName, percentage, FormatCurrency(oldAmount),
						FormatCurrency(scenario.CashFlow.Expenses[i].Amount)))
					break
				}
			}
		}
	}

	// Remove expense
	if opts.RemoveExpense != "" {
		for i, exp := range scenario.CashFlow.Expenses {
			if strings.EqualFold(exp.Name, opts.RemoveExpense) {
				changes = append(changes, fmt.Sprintf("Removed %s (was %s/month)",
					exp.Name, FormatCurrency(exp.Amount)))
				scenario.CashFlow.Expenses = append(scenario.CashFlow.Expenses[:i],
					scenario.CashFlow.Expenses[i+1:]...)
				break
			}
		}
	}

	// Add new expense
	if opts.AddExpense != "" {
		parts := strings.Split(opts.AddExpense, ":")
		if len(parts) == 2 {
			expenseName := parts[0]
			var amount float64
			fmt.Sscanf(parts[1], "%f", &amount)

			scenario.CashFlow.Expenses = append(scenario.CashFlow.Expenses, ExpenseItem{
				Name:        expenseName,
				Amount:      amount,
				DayOfMonth:  15,
				Type:        "variable",
				Essential:   false,
				Adjustable:  true,
				Flexibility: 0.5,
				Category:    "new",
			})
			changes = append(changes, fmt.Sprintf("Added %s (%s/month)",
				expenseName, FormatCurrency(amount)))
		}
	}

	// Apply income changes (similar logic)
	if opts.ReduceIncome != "" {
		parts := strings.Split(opts.ReduceIncome, ":")
		if len(parts) == 2 {
			incomeName := parts[0]
			var percentage float64
			fmt.Sscanf(parts[1], "%f", &percentage)

			for i := range scenario.CashFlow.Income {
				if strings.EqualFold(scenario.CashFlow.Income[i].Name, incomeName) {
					oldAmount := scenario.CashFlow.Income[i].Amount
					scenario.CashFlow.Income[i].Amount = oldAmount * (1 - percentage/100)
					changes = append(changes, fmt.Sprintf("Reduced %s by %.0f%% (%s → %s)",
						incomeName, percentage, FormatCurrency(oldAmount),
						FormatCurrency(scenario.CashFlow.Income[i].Amount)))
					break
				}
			}
		}
	}

	if opts.IncreaseIncome != "" {
		parts := strings.Split(opts.IncreaseIncome, ":")
		if len(parts) == 2 {
			incomeName := parts[0]
			var percentage float64
			fmt.Sscanf(parts[1], "%f", &percentage)

			for i := range scenario.CashFlow.Income {
				if strings.EqualFold(scenario.CashFlow.Income[i].Name, incomeName) {
					oldAmount := scenario.CashFlow.Income[i].Amount
					scenario.CashFlow.Income[i].Amount = oldAmount * (1 + percentage/100)
					changes = append(changes, fmt.Sprintf("Increased %s by %.0f%% (%s → %s)",
						incomeName, percentage, FormatCurrency(oldAmount),
						FormatCurrency(scenario.CashFlow.Income[i].Amount)))
					break
				}
			}
		}
	}

	if opts.AddIncome != "" {
		parts := strings.Split(opts.AddIncome, ":")
		if len(parts) == 2 {
			incomeName := parts[0]
			var amount float64
			fmt.Sscanf(parts[1], "%f", &amount)

			scenario.CashFlow.Income = append(scenario.CashFlow.Income, IncomeItem{
				Name:        incomeName,
				Amount:      amount,
				DayOfMonth:  1,
				Type:        "variable",
				Reliability: 0.8,
				Category:    "new",
			})
			changes = append(changes, fmt.Sprintf("Added %s (%s/month)",
				incomeName, FormatCurrency(amount)))
		}
	}

	// Extend goal timeline
	if opts.ExtendGoal != "" {
		parts := strings.Split(opts.ExtendGoal, ":")
		if len(parts) == 2 {
			goalName := parts[0]
			var months int
			fmt.Sscanf(parts[1], "%d", &months)

			for i := range scenario.Goals {
				if strings.EqualFold(scenario.Goals[i].Name, goalName) {
					oldTimeline := scenario.Goals[i].TimelineMonths
					scenario.Goals[i].TimelineMonths = oldTimeline + months
					changes = append(changes, fmt.Sprintf("Extended %s timeline by %d months (%d → %d)",
						goalName, months, oldTimeline, scenario.Goals[i].TimelineMonths))
					break
				}
			}
		}
	}

	// If no changes specified, show error
	if len(changes) == 0 {
		fmt.Println("⚠️  No changes specified. Use flags like:")
		fmt.Println("  --reduce-expense 'Dining:50'")
		fmt.Println("  --increase-income 'Salary:10'")
		fmt.Println("  --remove-expense 'Subscriptions'")
		return 0
	}

	// Calculate impacts
	baseIncome := config.CashFlow.CalculateMonthlyIncome()
	baseExpenses := config.CashFlow.CalculateMonthlyExpenses()
	baseSurplus := baseIncome - baseExpenses

	scenarioIncome := scenario.CashFlow.CalculateMonthlyIncome()
	scenarioExpenses := scenario.CashFlow.CalculateMonthlyExpenses()
	scenarioSurplus := scenarioIncome - scenarioExpenses

	surplusChange := scenarioSurplus - baseSurplus

	// Display results
	fmt.Println()
	fmt.Println(scenarioTitleStyle.Render(fmt.Sprintf("🔮 Scenario: %s", scenarioName)))
	fmt.Println()

	fmt.Println("Changes Made:")
	for _, change := range changes {
		fmt.Printf("  • %s\n", change)
	}
	fmt.Println()

	// Comparison table
	fmt.Println(scenarioTitleStyle.Render("Financial Impact:"))
	fmt.Printf("  %-20s %-15s %-15s %-15s\n", "Metric", "Current", "Scenario", "Change")
	fmt.Println(strings.Repeat("─", 70))

	fmt.Printf("  %-20s %-15s %-15s ", "Monthly Income",
		FormatCurrency(baseIncome), FormatCurrency(scenarioIncome))
	incomeChange := scenarioIncome - baseIncome
	if incomeChange > 0 {
		fmt.Printf("%+14s\n", positiveImpactStyle.Render(FormatCurrency(incomeChange)))
	} else if incomeChange < 0 {
		fmt.Printf("%+14s\n", negativeImpactStyle.Render(FormatCurrency(incomeChange)))
	} else {
		fmt.Printf("%+14s\n", neutralImpactStyle.Render("$0.00"))
	}

	fmt.Printf("  %-20s %-15s %-15s ", "Monthly Expenses",
		FormatCurrency(baseExpenses), FormatCurrency(scenarioExpenses))
	expenseChange := scenarioExpenses - baseExpenses
	if expenseChange < 0 {
		fmt.Printf("%+14s\n", positiveImpactStyle.Render(FormatCurrency(expenseChange)))
	} else if expenseChange > 0 {
		fmt.Printf("%+14s\n", negativeImpactStyle.Render(FormatCurrency(expenseChange)))
	} else {
		fmt.Printf("%+14s\n", neutralImpactStyle.Render("$0.00"))
	}

	fmt.Printf("  %-20s %-15s %-15s ", "Monthly Surplus",
		FormatCurrency(baseSurplus), FormatCurrency(scenarioSurplus))
	if surplusChange > 0 {
		fmt.Printf("%+14s\n", positiveImpactStyle.Render(FormatCurrency(surplusChange)))
	} else if surplusChange < 0 {
		fmt.Printf("%+14s\n", negativeImpactStyle.Render(FormatCurrency(surplusChange)))
	} else {
		fmt.Printf("%+14s\n", neutralImpactStyle.Render("$0.00"))
	}
	fmt.Println()

	// Goal impact
	if len(config.Goals) > 0 && surplusChange != 0 {
		fmt.Println(scenarioTitleStyle.Render("Goal Impact:"))

		for _, goal := range config.Goals {
			if goal.Target <= 0 {
				continue
			}

			remaining := goal.Target - goal.Current

			if baseSurplus > 0 {
				baseMonths := int(remaining / baseSurplus)
				if scenarioSurplus > 0 {
					scenarioMonths := int(remaining / scenarioSurplus)
					monthDiff := baseMonths - scenarioMonths

					fmt.Printf("  %s:\n", goal.Name)
					fmt.Printf("    Current trajectory: %d months\n", baseMonths)
					fmt.Printf("    New trajectory:     %d months", scenarioMonths)

					if monthDiff > 0 {
						fmt.Printf(" (%s faster)\n", positiveImpactStyle.Render(fmt.Sprintf("%d months", monthDiff)))
					} else if monthDiff < 0 {
						fmt.Printf(" (%s slower)\n", negativeImpactStyle.Render(fmt.Sprintf("%d months", -monthDiff)))
					} else {
						fmt.Println()
					}
				} else {
					fmt.Printf("  %s: %s\n", goal.Name,
						negativeImpactStyle.Render("⚠️  Cannot achieve - negative surplus"))
				}
			}
		}
		fmt.Println()
	}

	// Analysis
	fmt.Println(scenarioTitleStyle.Render("Analysis:"))

	if surplusChange > 0 {
		percentChange := (surplusChange / baseSurplus) * 100
		fmt.Printf("  ✅ Monthly surplus increases by %s (%.1f%%)\n",
			FormatCurrency(surplusChange), percentChange)

		if percentChange >= 20 {
			fmt.Printf("  %s\n", positiveImpactStyle.Render("  Significant improvement!"))
		}
	} else if surplusChange < 0 {
		fmt.Printf("  ⚠️  Monthly surplus decreases by %s\n", FormatCurrency(-surplusChange))

		if scenarioSurplus < 0 {
			fmt.Printf("  %s\n", negativeImpactStyle.Render("  ❌ WARNING: Budget becomes unsustainable!"))
			fmt.Printf("  %s\n", recommendationStyle.Render("  Recommendation: Reduce expenses further or increase income"))
		}
	} else {
		fmt.Println("  ℹ️  No change to monthly surplus")
	}

	// Savings rate comparison
	if baseIncome > 0 && scenarioIncome > 0 {
		baseRate := (baseSurplus / baseIncome) * 100
		scenarioRate := (scenarioSurplus / scenarioIncome) * 100

		fmt.Printf("\n  Savings rate: %.1f%% → %.1f%%\n", baseRate, scenarioRate)

		if scenarioRate >= 20 {
			fmt.Printf("  %s\n", positiveImpactStyle.Render("  ✅ Above recommended 20% savings rate"))
		} else if scenarioRate >= 10 {
			fmt.Printf("  %s\n", recommendationStyle.Render("  ⚠️  Below 20% recommendation"))
		} else {
			fmt.Printf("  %s\n", negativeImpactStyle.Render("  ❌ Low savings rate - increase savings"))
		}
	}

	fmt.Println()

	// Offer to save
	if scenarioSurplus > 0 {
		fmt.Printf("Would you like to save this scenario as '%s'? (y/N): ", scenarioName)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			scenarioPath := filepath.Join(GetConfigDir(), "scenarios", scenarioName+".yaml")

			data, err := yaml.Marshal(scenario)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to marshal scenario: %v\n", err)
				return 1
			}

			if err := os.WriteFile(scenarioPath, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to save scenario: %v\n", err)
				return 1
			}

			fmt.Printf("✅ Scenario saved to %s\n", scenarioPath)
		}
	}

	return 0
}

// cloneConfig creates a deep copy of the config
func cloneConfig(config *BudgetConfig) *BudgetConfig {
	// Simple approach: marshal to YAML and unmarshal
	data, _ := yaml.Marshal(config)
	var clone BudgetConfig
	yaml.Unmarshal(data, &clone)
	return &clone
}
