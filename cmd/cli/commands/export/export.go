package commands

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"voidline/internal/domain/timeutil"
	"voidline/internal/domain/transaction"
	"voidline/internal/infrastructure/persistence/sqlite"
)

// ExportCommand handles data export
type ExportCommand struct {
	DBPath    string
	Entity    string
	Format    string
	Output    string
	StartDate string
	EndDate   string
}

// Execute runs the export command
func (c *ExportCommand) Execute(ctx context.Context) error {
	// Connect to database
	conn, err := sqlite.NewConnection(c.DBPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	// Build filter
	filter := transaction.NewFilter().WithPerPage(100000)

	if c.StartDate != "" {
		date, err := timeutil.ParseDate(c.StartDate)
		if err != nil {
			return fmt.Errorf("invalid start date: %w", err)
		}
		filter.StartDate = &date
	}

	if c.EndDate != "" {
		date, err := timeutil.ParseDate(c.EndDate)
		if err != nil {
			return fmt.Errorf("invalid end date: %w", err)
		}
		filter.EndDate = &date
	}

	switch c.Entity {
	case "transactions":
		return c.exportTransactions(ctx, conn, filter)
	case "accounts":
		return c.exportAccounts(ctx, conn)
	default:
		return fmt.Errorf("unknown entity: %s", c.Entity)
	}
}

func (c *ExportCommand) exportTransactions(ctx context.Context, conn *sqlite.Connection, filter transaction.Filter) error {
	repo := sqlite.NewTransactionRepository(conn.DB())
	result, err := repo.FindByFilter(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query transactions: %w", err)
	}

	switch c.Format {
	case "json":
		return c.exportTransactionsJSON(result.Items)
	case "csv":
		return c.exportTransactionsCSV(result.Items)
	default:
		return fmt.Errorf("unknown format: %s", c.Format)
	}
}

func (c *ExportCommand) exportTransactionsJSON(transactions []transaction.Transaction) error {
	data, err := json.MarshalIndent(transactions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(c.Output, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (c *ExportCommand) exportTransactionsCSV(transactions []transaction.Transaction) error {
	file, err := os.Create(c.Output)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"id", "date", "name", "amount", "status", "category", "parent_category", "excluded", "tags", "type", "account", "account_mask", "note", "recurring"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write rows
	for _, tx := range transactions {
		excluded := "0"
		if tx.Excluded {
			excluded = "1"
		}

		recurring := "0"
		if tx.Recurring {
			recurring = "1"
		}

		row := []string{
			strconv.Itoa(tx.ID),
			tx.Date.String(),
			tx.Name,
			strconv.FormatFloat(tx.Amount, 'f', 2, 64),
			tx.Status,
			tx.Category,
			tx.ParentCategory,
			excluded,
			tx.Tags,
			tx.Type,
			tx.Account,
			tx.AccountMask,
			tx.Note,
			recurring,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

func (c *ExportCommand) exportAccounts(ctx context.Context, conn *sqlite.Connection) error {
	repo := sqlite.NewAccountRepository(conn.DB())
	accounts, err := repo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to query accounts: %w", err)
	}

	switch c.Format {
	case "json":
		data, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		if err := os.WriteFile(c.Output, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	case "csv":
		file, err := os.Create(c.Output)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		header := []string{"id", "name", "type", "active"}
		writer.Write(header)

		for _, acc := range accounts {
			active := "0"
			if acc.IsActive {
				active = "1"
			}
			row := []string{
				strconv.Itoa(acc.ID),
				acc.Name,
				string(acc.Type),
				active,
			}
			writer.Write(row)
		}
	default:
		return fmt.Errorf("unknown format: %s", c.Format)
	}

	return nil
}
