package services

import (
	"context"
	"testing"

	"voidline/internal/testutil/testdb"
	"github.com/stretchr/testify/assert"
)

func TestTransactionsServiceWithDB(t *testing.T) {
	db := testdb.NewTestDB(t)

	txRepo := newTestTransactionRepository(db)
	svc := NewFinanceTransactionsService(txRepo)

	t.Run("CreateTransaction", func(t *testing.T) {
		tx, err := svc.CreateTransaction(context.Background(), CreateTransactionInput{
			Date:     "2024-01-15",
			Name:     "Grocery Store",
			Amount:   125.50,
			Account:  "Checking",
			Category: "Food",
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, tx.ID)
		assert.Equal(t, "Grocery Store", tx.Payee)
		assert.Equal(t, 125.50, tx.Amount)
	})

	t.Run("GetTransactions", func(t *testing.T) {
		svc.CreateTransaction(context.Background(), CreateTransactionInput{
			Date:    "2024-01-16",
			Name:    "Coffee Shop",
			Amount:  5.50,
			Account: "Checking",
		})

		txs, err := svc.GetTransactions(context.Background(), GetTransactionsOptions{
			Account: "Checking",
			PerPage: 10,
			Page:    1,
		})

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(txs), 1)
	})

	t.Run("UpdateTransaction", func(t *testing.T) {
		tx, err := svc.CreateTransaction(context.Background(), CreateTransactionInput{
			Date:    "2024-01-17",
			Name:    "Restaurant",
			Amount:  45.00,
			Account: "Checking",
		})
		assert.NoError(t, err)

		updated, err := svc.UpdateTransaction(context.Background(), UpdateTransactionInput{
			ID:     tx.ID,
			Amount: 50.00,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, updated.ID)
	})

	t.Run("DeleteTransaction", func(t *testing.T) {
		tx, err := svc.CreateTransaction(context.Background(), CreateTransactionInput{
			Date:    "2024-01-18",
			Name:    "To Delete",
			Amount:  10.00,
			Account: "Checking",
		})
		assert.NoError(t, err)

		err = svc.DeleteTransaction(context.Background(), tx.ID)
		assert.NoError(t, err)

		txs, _ := svc.GetTransactions(context.Background(), GetTransactionsOptions{
			PerPage: 100,
			Page:    1,
		})
		for _, tr := range txs {
			assert.NotEqual(t, "To Delete", tr.Payee)
		}
	})

	t.Run("Validation", func(t *testing.T) {
		_, err := svc.CreateTransaction(context.Background(), CreateTransactionInput{
			Date:    "",
			Name:    "Test",
			Amount:  100,
			Account: "Checking",
		})
		assert.Error(t, err)
	})

	t.Run("CreateTransactionsBatch", func(t *testing.T) {
		result, err := svc.CreateTransactionsBatch(context.Background(), CreateTransactionsBatchInput{
			Transactions: []CreateTransactionInput{
				{Date: "2024-01-20", Name: "Batch Tx 1", Amount: 100, Account: "Checking"},
				{Date: "2024-01-21", Name: "Batch Tx 2", Amount: 200, Account: "Checking"},
				{Date: "2024-01-22", Name: "Batch Tx 3", Amount: 300, Account: "Checking"},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, result.Created)
		assert.Equal(t, 0, result.Skipped)
	})

	t.Run("CreateTransactionsBatch with skip duplicates", func(t *testing.T) {
		result, err := svc.CreateTransactionsBatch(context.Background(), CreateTransactionsBatchInput{
			Transactions: []CreateTransactionInput{
				{Date: "2024-01-23", Name: "Batch Dup 1", Amount: 100, Account: "Checking"},
				{Date: "2024-01-24", Name: "Batch Dup 2", Amount: 150, Account: "Checking"},
			},
			SkipDuplicates: true,
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, result.Created)
	})
}

func TestAccountsServiceWithDB(t *testing.T) {
	db := testdb.NewTestDB(t)

	accRepo := newTestAccountRepository(db)
	txRepo := newTestTransactionRepository(db)
	svc := NewFinanceAccountsService(accRepo, txRepo)

	t.Run("CreateAccount", func(t *testing.T) {
		acc, err := svc.CreateAccount(context.Background(), CreateAccountInput{
			Name:     "Test Checking",
			Type:     "CHECKING",
			Currency: "USD",
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, acc.ID)
		assert.Equal(t, "Test Checking", acc.Name)
		assert.Equal(t, "CHECKING", acc.Type)
	})

	t.Run("GetAccounts", func(t *testing.T) {
		svc.CreateAccount(context.Background(), CreateAccountInput{
			Name:     "Savings Account",
			Type:     "SAVINGS",
			Currency: "USD",
		})

		accounts, err := svc.GetAccounts(context.Background())
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(accounts), 1)
	})

	t.Run("UpdateAccount", func(t *testing.T) {
		acc, _ := svc.CreateAccount(context.Background(), CreateAccountInput{
			Name:     "Original Name",
			Type:     "CHECKING",
			Currency: "USD",
		})

		updated, err := svc.UpdateAccount(context.Background(), UpdateAccountInput{
			ID:   acc.ID,
			Name: "Updated Name",
		})
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
	})

	t.Run("DeleteAccount", func(t *testing.T) {
		acc, _ := svc.CreateAccount(context.Background(), CreateAccountInput{
			Name:     "To Delete",
			Type:     "CHECKING",
			Currency: "USD",
		})

		txRepo := newTestTransactionRepository(db)
		txSvc := NewFinanceTransactionsService(txRepo)
		txSvc.CreateTransaction(context.Background(), CreateTransactionInput{
			Date:    "2024-01-15",
			Name:    "Test",
			Amount:  100,
			Account: "To Delete",
		})

		count, err := svc.DeleteAccount(context.Background(), acc.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestCategoriesServiceWithDB(t *testing.T) {
	db := testdb.NewTestDB(t)

	catRepo := newTestCategoryRepository(db)
	svc := NewFinanceCategoriesService(catRepo)

	t.Run("CreateCategory", func(t *testing.T) {
		cat, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
			Name:   "Groceries",
			Domain: "finance",
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, cat.ID)
		assert.Equal(t, "Groceries", cat.Name)
	})

	t.Run("GetCategories", func(t *testing.T) {
		svc.CreateCategory(context.Background(), CreateCategoryInput{
			Name:   "Entertainment",
			Domain: "finance",
		})

		categories, err := svc.GetCategories(context.Background(), "finance")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(categories), 1)
	})

	t.Run("Validation", func(t *testing.T) {
		_, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
			Name:   "",
			Domain: "finance",
		})
		assert.Error(t, err)
	})

	t.Run("CategoryHierarchy", func(t *testing.T) {
		// Create parent category with unique name
		parent, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
			Name:   "Food & Dining",
			Domain: "finance",
		})
		assert.NoError(t, err)

		// Create child category with parent ID
		parentIDStr := parent.ID
		child, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
			Name:     "Fresh Produce",
			ParentID: &parentIDStr,
			Domain:   "finance",
		})
		assert.NoError(t, err)
		assert.NotNil(t, child.ParentID)
		assert.Equal(t, parentIDStr, *child.ParentID)

		// Get tree and verify hierarchy
		tree, err := svc.GetCategoryTree(context.Background(), "finance")
		assert.NoError(t, err)

		// Find parent in tree
		var foundParent *FinanceCategoryDTO
		for i := range tree {
			if tree[i].ID == parentIDStr {
				foundParent = &tree[i]
				break
			}
		}

		assert.NotNil(t, foundParent)
		assert.GreaterOrEqual(t, len(foundParent.Children), 1)

		// Verify child is in parent's children
		var foundChild *FinanceCategoryDTO
		for i := range foundParent.Children {
			if foundParent.Children[i].ID == child.ID {
				foundChild = &foundParent.Children[i]
				break
			}
		}
		assert.NotNil(t, foundChild)
	})

	t.Run("CategoryDeletion", func(t *testing.T) {
		// Create a category to delete
		cat1, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
			Name:   "To Be Deleted",
			Domain: "finance",
		})
		assert.NoError(t, err)

		// Verify category exists
		cats1, _ := svc.GetCategories(context.Background(), "finance")
		found := false
		for _, c := range cats1 {
			if c.ID == cat1.ID {
				found = true
				break
			}
		}
		assert.True(t, found)

		// Delete category (should return success or result)
		result, err := svc.DeleteCategory(context.Background(), cat1.ID)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify category is deleted
		cats2, _ := svc.GetCategories(context.Background(), "finance")
		for _, c := range cats2 {
			assert.NotEqual(t, cat1.ID, c.ID)
		}
	})
}
