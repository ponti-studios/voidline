package transaction

import (
	"fmt"
	"time"

	"voidline/internal/domain/timeutil"
)

// Transaction represents a financial transaction entity
type Transaction struct {
	ID             int
	Date           timeutil.Date
	Name           string
	Amount         float64
	Status         string
	Category       string
	ParentCategory string
	Excluded       bool
	Tags           string
	Type           string
	Account        string
	AccountMask    string
	Note           string
	Recurring      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsExpense returns true if this transaction is an expense (negative amount)
func (t *Transaction) IsExpense() bool {
	return t.Amount < 0
}

// IsIncome returns true if this transaction is income (positive amount)
func (t *Transaction) IsIncome() bool {
	return t.Amount > 0
}

// AbsAmount returns the absolute value of the amount
func (t *Transaction) AbsAmount() float64 {
	if t.Amount < 0 {
		return -t.Amount
	}
	return t.Amount
}

// Key generates a unique key for deduplication
func (t *Transaction) Key() string {
	return t.Date.String() + "|" + t.Name + "|" + t.Account + "|" + formatAmount(t.Amount)
}

func formatAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}
