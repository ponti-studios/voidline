package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"voidline/internal/domain/timeutil"
	"voidline/internal/domain/transaction"
)

// Reader handles CSV file reading for transactions
type Reader struct {
	dateFormat string
}

// NewReader creates a new CSV reader
func NewReader() *Reader {
	return &Reader{
		dateFormat: "2006-01-02",
	}
}

// NewReaderWithDateFormat creates a new CSV reader with a custom date format
func NewReaderWithDateFormat(dateFormat string) *Reader {
	return &Reader{
		dateFormat: dateFormat,
	}
}

// Transaction represents a row from a CSV file before conversion to domain model
type Transaction struct {
	Date           string
	Name           string
	Amount         string
	Status         string
	Category       string
	ParentCategory string
	Excluded       string
	Tags           string
	Type           string
	Account        string
	AccountMask    string
	Note           string
	Recurring      string
}

// ReadTransactions reads all transactions from a CSV reader
func (r *Reader) ReadTransactions(reader io.Reader) ([]Transaction, error) {
	csvReader := csv.NewReader(reader)

	// Read header
	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Map headers to indices
	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var transactions []Transaction
	lineNum := 1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading line %d: %w", lineNum, err)
		}

		lineNum++
		transactions = append(transactions, r.parseRow(record, colMap))
	}

	return transactions, nil
}

// ReadAndConvert reads CSV and converts to domain transactions
func (r *Reader) ReadAndConvert(reader io.Reader) ([]transaction.Transaction, []error) {
	csvReader := csv.NewReader(reader)

	// Read header
	headers, err := csvReader.Read()
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read CSV header: %w", err)}
	}

	// Map headers to indices
	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var transactions []transaction.Transaction
	var errors []error
	lineNum := 1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Errorf("error reading line %d: %w", lineNum, err))
			lineNum++
			continue
		}

		lineNum++
		tx, err := r.convertRow(record, colMap)
		if err != nil {
			errors = append(errors, fmt.Errorf("line %d: %w", lineNum, err))
			continue
		}

		transactions = append(transactions, tx)
	}

	return transactions, errors
}

func (r *Reader) parseRow(record []string, colMap map[string]int) Transaction {
	return Transaction{
		Date:           r.getField(record, colMap, "date"),
		Name:           r.getField(record, colMap, "name"),
		Amount:         r.getField(record, colMap, "amount"),
		Status:         r.getField(record, colMap, "status"),
		Category:       r.getField(record, colMap, "category"),
		ParentCategory: r.getField(record, colMap, "parent_category", "parent category"),
		Excluded:       r.getField(record, colMap, "excluded"),
		Tags:           r.getField(record, colMap, "tags"),
		Type:           r.getField(record, colMap, "type"),
		Account:        r.getField(record, colMap, "account", "accounts"),
		AccountMask:    r.getField(record, colMap, "account_mask", "account mask"),
		Note:           r.getField(record, colMap, "note"),
		Recurring:      r.getField(record, colMap, "recurring"),
	}
}

func (r *Reader) convertRow(record []string, colMap map[string]int) (transaction.Transaction, error) {
	csvTx := r.parseRow(record, colMap)

	// Parse date
	date, err := timeutil.ParseDate(csvTx.Date)
	if err != nil {
		return transaction.Transaction{}, fmt.Errorf("invalid date '%s': %w", csvTx.Date, err)
	}

	// Parse amount
	amount, err := strconv.ParseFloat(strings.TrimSpace(csvTx.Amount), 64)
	if err != nil {
		return transaction.Transaction{}, fmt.Errorf("invalid amount '%s': %w", csvTx.Amount, err)
	}

	// Parse excluded
	excluded := strings.ToLower(csvTx.Excluded) == "true" || csvTx.Excluded == "1"

	// Parse recurring
	recurring := strings.ToLower(csvTx.Recurring) == "true" || csvTx.Recurring == "1"

	return transaction.Transaction{
		Date:           date,
		Name:           strings.TrimSpace(csvTx.Name),
		Amount:         amount,
		Status:         strings.TrimSpace(csvTx.Status),
		Category:       strings.TrimSpace(csvTx.Category),
		ParentCategory: strings.TrimSpace(csvTx.ParentCategory),
		Excluded:       excluded,
		Tags:           csvTx.Tags,
		Type:           csvTx.Type,
		Account:        strings.TrimSpace(csvTx.Account),
		AccountMask:    csvTx.AccountMask,
		Note:           csvTx.Note,
		Recurring:      recurring,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (r *Reader) getField(record []string, colMap map[string]int, keys ...string) string {
	for _, key := range keys {
		if idx, ok := colMap[key]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
	}
	return ""
}
