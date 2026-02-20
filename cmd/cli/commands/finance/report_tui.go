package finance

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"voidline/internal/application/services"
)

var (
	listTitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE082")).Bold(true)
	detailBoxStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1).Width(60)
	emptyDetailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
)

// generic list item wrappers -------------------------------------------------
type txItem struct {
	tx services.FinanceTransactionDTO
}

func (i txItem) Title() string { return fmt.Sprintf("%s — %s", i.tx.Date, i.tx.Payee) }
func (i txItem) Description() string {
	return fmt.Sprintf("%s · %s · %.2f", i.tx.Category, i.tx.Account, i.tx.Amount)
}
func (i txItem) FilterValue() string {
	return strings.Join([]string{i.tx.Payee, i.tx.Category, i.tx.Account}, " ")
}

type accItem struct{ a services.FinanceAccountDTO }

func (i accItem) Title() string { return i.a.Name }
func (i accItem) Description() string {
	return fmt.Sprintf("%s · %.2f %s", i.a.Type, i.a.Balance, i.a.Currency)
}
func (i accItem) FilterValue() string { return strings.Join([]string{i.a.Name, i.a.Type}, " ") }

type catItem struct{ c services.FinanceCategoryDTO }

func (i catItem) Title() string       { return i.c.Name }
func (i catItem) Description() string { return i.c.Domain }
func (i catItem) FilterValue() string { return strings.Join([]string{i.c.Name, i.c.Domain}, " ") }

// transactions list TUI ------------------------------------------------------
type transactionsModel struct {
	l   list.Model
	txs []services.FinanceTransactionDTO
}

func (m transactionsModel) Init() tea.Cmd { return nil }

func (m transactionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.l, cmd = m.l.Update(msg)
	return m, cmd
}

func (m transactionsModel) View() string {
	b := &strings.Builder{}
	b.WriteString(listTitleStyle.Render(" Transactions ") + "\n\n")
	b.WriteString(m.l.View())

	if sel := m.l.SelectedItem(); sel != nil {
		item := sel.(txItem)
		details := fmt.Sprintf("Date: %s\nPayee: %s\nAccount: %s\nCategory: %s\nAmount: %.2f\nNotes: %s",
			item.tx.Date, item.tx.Payee, item.tx.Account, item.tx.Category, item.tx.Amount, item.tx.Notes)
		b.WriteString("\n" + detailBoxStyle.Render(details))
	} else {
		b.WriteString("\n" + emptyDetailStyle.Render("No transaction selected"))
	}

	b.WriteString("\n\nPress q or Esc to quit. Use / to filter.")
	return b.String()
}

func runTransactionsTUI(txs []services.FinanceTransactionDTO) error {
	items := make([]list.Item, 0, len(txs))
	for _, t := range txs {
		items = append(items, txItem{tx: t})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 14)
	l.Title = "Transactions"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.KeyMap.Quit.SetEnabled(false)

	m := transactionsModel{l: l, txs: txs}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// accounts list TUI ----------------------------------------------------------
type accountsModel struct {
	l    list.Model
	accs []services.FinanceAccountDTO
}

func (m accountsModel) Init() tea.Cmd { return nil }

func (m accountsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.l, cmd = m.l.Update(msg)
	return m, cmd
}

func (m accountsModel) View() string {
	b := &strings.Builder{}
	b.WriteString(listTitleStyle.Render(" Accounts ") + "\n\n")
	b.WriteString(m.l.View())

	if sel := m.l.SelectedItem(); sel != nil {
		item := sel.(accItem)
		details := fmt.Sprintf("Name: %s\nType: %s\nBalance: %.2f %s\nActive: %t\nLastUpdated: %s",
			item.a.Name, item.a.Type, item.a.Balance, item.a.Currency, item.a.IsActive, item.a.LastUpdated)
		b.WriteString("\n" + detailBoxStyle.Render(details))
	} else {
		b.WriteString("\n" + emptyDetailStyle.Render("No account selected"))
	}

	b.WriteString("\n\nPress q or Esc to quit. Use / to filter.")
	return b.String()
}

func runAccountsTUI(accs []services.FinanceAccountDTO) error {
	items := make([]list.Item, 0, len(accs))
	for _, a := range accs {
		items = append(items, accItem{a: a})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 12)
	l.Title = "Accounts"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	m := accountsModel{l: l, accs: accs}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// categories list TUI -------------------------------------------------------
type categoriesModel struct {
	l    list.Model
	cats []services.FinanceCategoryDTO
}

func (m categoriesModel) Init() tea.Cmd { return nil }

func (m categoriesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.l, cmd = m.l.Update(msg)
	return m, cmd
}

func (m categoriesModel) View() string {
	b := &strings.Builder{}
	b.WriteString(listTitleStyle.Render(" Categories ") + "\n\n")
	b.WriteString(m.l.View())

	if sel := m.l.SelectedItem(); sel != nil {
		item := sel.(catItem)
		detail := fmt.Sprintf("Name: %s\nDomain: %s\nDescription: %s", item.c.Name, item.c.Domain, item.c.Description)
		b.WriteString("\n" + detailBoxStyle.Render(detail))
	} else {
		b.WriteString("\n" + emptyDetailStyle.Render("No category selected"))
	}

	b.WriteString("\n\nPress q or Esc to quit. Use / to filter.")
	return b.String()
}

func runCategoriesTUI(cats []services.FinanceCategoryDTO) error {
	items := make([]list.Item, 0, len(cats))
	for _, c := range cats {
		items = append(items, catItem{c: c})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 12)
	l.Title = "Categories"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	m := categoriesModel{l: l, cats: cats}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
