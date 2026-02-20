package importservice

import (
	"fmt"
	"strings"

	"voidline/internal/domain/transaction"
)

// Validator handles validation of imported transactions
type Validator struct{}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate validates a transaction
func (v *Validator) Validate(tx transaction.Transaction) error {
	if tx.Date.IsZero() {
		return ValidationError{Field: "date", Message: "date is required"}
	}

	if strings.TrimSpace(tx.Name) == "" {
		return ValidationError{Field: "name", Message: "name is required"}
	}

	if strings.TrimSpace(tx.Account) == "" {
		return ValidationError{Field: "account", Message: "account is required"}
	}

	if strings.TrimSpace(tx.Category) == "" {
		return ValidationError{Field: "category", Message: "category is required"}
	}

	return nil
}

// ValidateAmount validates the amount field
func (v *Validator) ValidateAmount(amount float64) error {
	// Amount can be positive (income) or negative (expense)
	// Zero is allowed for certain transaction types
	return nil
}

// ValidateDate validates the date field
func (v *Validator) ValidateDate(date string) error {
	if date == "" {
		return ValidationError{Field: "date", Message: "date is required"}
	}
	return nil
}
