package amazon

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"voidline/internal/domain/amazon"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type Service struct {
	repo *sqlite.AmazonRepository
}

func NewService(repo *sqlite.AmazonRepository) *Service {
	return &Service{repo: repo}
}

type ImportOptions struct {
	DryRun bool
	Force  bool
}

func (s *Service) ImportOrderHistory(ctx context.Context, sourceDir string, options ImportOptions) (*amazon.ImportResult, error) {
	pattern := sourceDir + "/amazon/Retail.OrderHistory.*/*.csv"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob pattern: %w", err)
	}

	if len(matches) == 0 {
		fmt.Printf("No Amazon CSV files found matching %s\n", pattern)
		return &amazon.ImportResult{}, nil
	}

	fmt.Printf("Found %d CSV files.\n", len(matches))

	existing, err := s.repo.LoadExisting(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing orders: %w", err)
	}

	totalInserted := 0
	totalSkipped := 0

	for _, csvFile := range matches {
		fmt.Printf("Processing %s...\n", filepath.Base(csvFile))
		inserted, skipped, err := s.processFile(ctx, csvFile, existing, options)
		if err != nil {
			fmt.Printf("  Error processing file: %v\n", err)
			continue
		}
		totalInserted += inserted
		totalSkipped += skipped
	}

	fmt.Printf("Total Inserted: %d\n", totalInserted)
	fmt.Printf("Total Skipped: %d\n", totalSkipped)

	return &amazon.ImportResult{
		TotalRows: totalInserted + totalSkipped,
		Inserted:  totalInserted,
		Skipped:   totalSkipped,
	}, nil
}

func (s *Service) processFile(ctx context.Context, filePath string, existing map[string]bool, options ImportOptions) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read header: %w", err)
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.TrimSpace(h)] = i
	}

	var toInsert []interface{}
	inserted := 0
	skipped := 0

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		orderID := getField(record, colMap, "Order ID")
		asin := getField(record, colMap, "ASIN")
		key := orderID + "|" + asin

		if !options.Force && existing[key] {
			skipped++
			continue
		}

		orderDate := getField(record, colMap, "Order Date")
		title := getField(record, colMap, "Product Name")
		unitPrice := parseFloat(getField(record, colMap, "Unit Price"))
		quantity := parseInt(getField(record, colMap, "Quantity"))
		shipDate := getField(record, colMap, "Ship Date")
		fullAddress := getField(record, colMap, "Shipping Address")
		orderStatus := getField(record, colMap, "Order Status")
		carrier := getField(record, colMap, "Carrier Name & Tracking Number")
		subtotal := parseFloat(getField(record, colMap, "Shipment Item Subtotal"))
		tax := parseFloat(getField(record, colMap, "Shipment Item Subtotal Tax"))
		itemTotal := subtotal + tax

		toInsert = append(toInsert, []interface{}{
			orderDate, orderID, title, nil, asin,
			unitPrice, quantity, shipDate,
			nil, fullAddress, nil,
			nil, nil, nil,
			orderStatus, carrier, subtotal, tax, itemTotal,
		})

		existing[key] = true
		inserted++
	}

	if !options.DryRun && len(toInsert) > 0 {
		if err := s.repo.InsertBatch(ctx, toInsert); err != nil {
			return inserted, skipped, fmt.Errorf("failed to insert batch: %w", err)
		}
		fmt.Printf("  Inserted %d records.\n", inserted)
	} else if options.DryRun {
		fmt.Printf("  Would insert %d records (dry run).\n", inserted)
	}

	return inserted, skipped, nil
}

func getField(record []string, colMap map[string]int, keys ...string) string {
	for _, key := range keys {
		if idx, ok := colMap[key]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
	}
	return ""
}

func parseFloat(val string) float64 {
	if val == "" {
		return 0.0
	}
	val = strings.ReplaceAll(val, "$", "")
	val = strings.ReplaceAll(val, ",", "")
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0
	}
	return f
}

func parseInt(val string) int {
	if val == "" {
		return 0
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}
