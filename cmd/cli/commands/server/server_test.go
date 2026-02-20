package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"voidline/cmd/cli/commands/server/handlers"
	"voidline/internal/application/services"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type ServerTestSuite struct {
	suite.Suite
	router          *gin.Engine
	testServer      *httptest.Server
	dbPath          string
	db              *sqlite.Connection
	accountsURL     string
	transactionsURL string
	categoriesURL   string
	dashboardURL    string
}

func (s *ServerTestSuite) SetupSuite() {
	dbPath := "/tmp/test_hominem_" + randomString(8) + ".sqlite3"
	s.dbPath = dbPath

	db, err := sqlite.NewConnection(dbPath)
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	s.db = db

	// Create required tables for tests
	schema := `
	CREATE TABLE IF NOT EXISTS financial_accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		type TEXT,
		credit_limit REAL,
		active INTEGER
	);

	CREATE TABLE IF NOT EXISTS finance_transactions (
		id INTEGER PRIMARY KEY,
		date TEXT NOT NULL,
		name TEXT NOT NULL,
		amount REAL NOT NULL,
		status TEXT NOT NULL,
		category TEXT NOT NULL,
		parent_category TEXT NOT NULL,
		excluded INTEGER DEFAULT 0,
		tags TEXT,
		type TEXT NOT NULL,
		account TEXT NOT NULL,
		account_mask TEXT,
		note TEXT,
		recurring INTEGER,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER,
		domain TEXT NOT NULL,
		description TEXT,
		is_active INTEGER DEFAULT 1,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.DB().Exec(schema)
	if err != nil {
		panic("failed to create test schema: " + err.Error())
	}

	accountRepo := sqlite.NewAccountRepository(db.DB())
	transactionRepo := sqlite.NewTransactionRepository(db.DB())
	categoryRepo := sqlite.NewCategoryRepository(db.DB())

	accountsService := services.NewFinanceAccountsService(accountRepo, transactionRepo)
	transactionsService := services.NewFinanceTransactionsService(transactionRepo)
	categoriesService := services.NewFinanceCategoriesService(categoryRepo)
	dashboardService := services.NewFinanceDashboardService(accountRepo)

	accountsHandler := handlers.NewAccountsHandler(accountsService)
	transactionsHandler := handlers.NewTransactionsHandler(transactionsService)
	categoriesHandler := handlers.NewCategoriesHandler(categoriesService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	s.router = gin.New()

	api := s.router.Group("/api/v1")
	accountsHandler.RegisterRoutes(api)
	transactionsHandler.RegisterRoutes(api)
	categoriesHandler.RegisterRoutes(api)
	dashboardHandler.RegisterRoutes(api)

	s.testServer = httptest.NewServer(s.router)
	s.accountsURL = s.testServer.URL + "/api/v1/accounts"
	s.transactionsURL = s.testServer.URL + "/api/v1/transactions"
	s.categoriesURL = s.testServer.URL + "/api/v1/categories"
	s.dashboardURL = s.testServer.URL + "/api/v1/dashboard"
}

func (s *ServerTestSuite) TearDownSuite() {
	s.testServer.Close()
	if s.db != nil {
		s.db.Close()
	}
	os.Remove(s.dbPath)
}

func (s *ServerTestSuite) TestDashboardEmpty() {
	resp, err := http.Get(s.dashboardURL)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	s.NoError(err)

	s.Contains(data, "totalBalance")
	s.Contains(data, "accounts")
}

func (s *ServerTestSuite) TestCreateAndGetAccount() {
	account := map[string]string{
		"name":     "Test Checking",
		"type":     "CHECKING",
		"currency": "USD",
	}
	body, _ := json.Marshal(account)

	resp, err := http.Post(s.accountsURL, "application/json", bytes.NewReader(body))
	s.NoError(err)
	s.Equal(http.StatusCreated, resp.StatusCode)

	var created map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&created)
	s.NoError(err)
	s.Equal("Test Checking", created["name"])
	s.Equal("CHECKING", created["type"])
	s.Contains(created, "id")

	id := created["id"].(string)

	resp, err = http.Get(s.accountsURL + "/" + id)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var fetched map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&fetched)
	s.NoError(err)
	s.Equal("Test Checking", fetched["name"])
}

func (s *ServerTestSuite) TestGetAccounts() {
	resp, err := http.Get(s.accountsURL)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var accounts []interface{}
	err = json.NewDecoder(resp.Body).Decode(&accounts)
	s.NoError(err)
	s.True(len(accounts) > 0)
}

func (s *ServerTestSuite) TestUpdateAccount() {
	account := map[string]string{
		"name":     "Update Test Account",
		"type":     "SAVINGS",
		"currency": "USD",
	}
	body, _ := json.Marshal(account)

	resp, err := http.Post(s.accountsURL, "application/json", bytes.NewReader(body))
	s.NoError(err)
	s.Equal(http.StatusCreated, resp.StatusCode)

	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	id := created["id"].(string)

	update := map[string]string{"name": "Updated Name"}
	updateBody, _ := json.Marshal(update)

	req, _ := http.NewRequest(http.MethodPut, s.accountsURL+"/"+id, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var updated map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&updated)
	s.Equal("Updated Name", updated["name"])
}

func (s *ServerTestSuite) TestCreateTransaction() {
	account := map[string]string{
		"name":     "Transaction Test Account",
		"type":     "CHECKING",
		"currency": "USD",
	}
	accBody, _ := json.Marshal(account)
	http.Post(s.accountsURL, "application/json", bytes.NewReader(accBody))

	transaction := map[string]interface{}{
		"name":    "Test Transaction",
		"amount":  100.50,
		"account": "Transaction Test Account",
		"date":    "2024-01-15",
	}
	txBody, _ := json.Marshal(transaction)

	resp, err := http.Post(s.transactionsURL, "application/json", bytes.NewReader(txBody))
	s.NoError(err)
	s.Equal(http.StatusCreated, resp.StatusCode)

	var created map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&created)
	s.NoError(err)
	s.Equal("Test Transaction", created["payee"])
	s.Equal(100.50, created["amount"])
}

func (s *ServerTestSuite) TestGetTransactions() {
	resp, err := http.Get(s.transactionsURL)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var transactions []interface{}
	err = json.NewDecoder(resp.Body).Decode(&transactions)
	s.NoError(err)
	s.True(len(transactions) >= 0)
}

func (s *ServerTestSuite) TestUpdateTransaction() {
	account := map[string]string{
		"name":     "Update Tx Account",
		"type":     "CHECKING",
		"currency": "USD",
	}
	accBody, _ := json.Marshal(account)
	http.Post(s.accountsURL, "application/json", bytes.NewReader(accBody))

	transaction := map[string]interface{}{
		"name":    "Original Transaction",
		"amount":  50.00,
		"account": "Update Tx Account",
		"date":    "2024-01-20",
	}
	txBody, _ := json.Marshal(transaction)

	resp, err := http.Post(s.transactionsURL, "application/json", bytes.NewReader(txBody))
	s.NoError(err)

	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	id := created["id"].(string)

	update := map[string]interface{}{"name": "Updated Transaction"}
	updateBody, _ := json.Marshal(update)

	req, _ := http.NewRequest(http.MethodPut, s.transactionsURL+"/"+id, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var updated map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&updated)
	s.Equal("Updated Transaction", updated["payee"])
}

func (s *ServerTestSuite) TestDeleteTransaction() {
	account := map[string]string{
		"name":     "Delete Tx Account",
		"type":     "CHECKING",
		"currency": "USD",
	}
	accBody, _ := json.Marshal(account)
	http.Post(s.accountsURL, "application/json", bytes.NewReader(accBody))

	transaction := map[string]interface{}{
		"name":    "To Be Deleted",
		"amount":  25.00,
		"account": "Delete Tx Account",
		"date":    "2024-01-25",
	}
	txBody, _ := json.Marshal(transaction)

	resp, err := http.Post(s.transactionsURL, "application/json", bytes.NewReader(txBody))
	s.NoError(err)

	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	id := created["id"].(string)

	req, _ := http.NewRequest(http.MethodDelete, s.transactionsURL+"/"+id, nil)
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *ServerTestSuite) TestCreateCategory() {
	category := map[string]string{
		"name":   "Test Category",
		"domain": "finance",
	}
	body, _ := json.Marshal(category)

	resp, err := http.Post(s.categoriesURL, "application/json", bytes.NewReader(body))
	s.NoError(err)
	s.Equal(http.StatusCreated, resp.StatusCode)

	var created map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&created)
	s.NoError(err)
	s.Equal("Test Category", created["name"])
	s.Equal("finance", created["domain"])
}

func (s *ServerTestSuite) TestGetCategories() {
	resp, err := http.Get(s.categoriesURL)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var categories []interface{}
	err = json.NewDecoder(resp.Body).Decode(&categories)
	s.NoError(err)
	s.True(len(categories) >= 0)
}

func (s *ServerTestSuite) TestGetCategoryTree() {
	category := map[string]string{
		"name":   "Tree Parent",
		"domain": "finance",
	}
	body, _ := json.Marshal(category)
	http.Post(s.categoriesURL, "application/json", bytes.NewReader(body))

	resp, err := http.Get(s.categoriesURL + "/tree")
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var tree []interface{}
	err = json.NewDecoder(resp.Body).Decode(&tree)
	s.NoError(err)
	s.NotNil(tree)
}

func (s *ServerTestSuite) TestUpdateCategory() {
	category := map[string]string{
		"name":   "Category To Update",
		"domain": "finance",
	}
	body, _ := json.Marshal(category)

	resp, err := http.Post(s.categoriesURL, "application/json", bytes.NewReader(body))
	s.NoError(err)

	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	id := created["id"].(string)

	update := map[string]string{"name": "Updated Category Name"}
	updateBody, _ := json.Marshal(update)

	req, _ := http.NewRequest(http.MethodPut, s.categoriesURL+"/"+id, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	var updated map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&updated)
	s.Equal("Updated Category Name", updated["name"])
}

func (s *ServerTestSuite) TestDeleteCategory() {
	category := map[string]string{
		"name":   "Category To Delete",
		"domain": "finance",
	}
	body, _ := json.Marshal(category)

	resp, err := http.Post(s.categoriesURL, "application/json", bytes.NewReader(body))
	s.NoError(err)

	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	id := created["id"].(string)

	req, _ := http.NewRequest(http.MethodDelete, s.categoriesURL+"/"+id, nil)
	resp, err = http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ServerTestSuite) TestBatchCreateTransactions() {
	account := map[string]string{
		"name":     "Batch Account",
		"type":     "CHECKING",
		"currency": "USD",
	}
	accBody, _ := json.Marshal(account)
	http.Post(s.accountsURL, "application/json", bytes.NewReader(accBody))

	batch := map[string]interface{}{
		"transactions": []map[string]interface{}{
			{"name": "Batch 1", "amount": 10.0, "account": "Batch Account", "date": "2024-02-01"},
			{"name": "Batch 2", "amount": 20.0, "account": "Batch Account", "date": "2024-02-02"},
			{"name": "Batch 3", "amount": 30.0, "account": "Batch Account", "date": "2024-02-03"},
		},
		"skipDuplicates": false,
	}
	batchBody, _ := json.Marshal(batch)

	resp, err := http.Post(s.transactionsURL+"/batch", "application/json", bytes.NewReader(batchBody))
	s.NoError(err)
	s.Equal(http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal(float64(3), result["created"])
}

func (s *ServerTestSuite) TestValidationErrors() {
	invalid := map[string]string{
		"name": "",
	}
	body, _ := json.Marshal(invalid)

	resp, err := http.Post(s.accountsURL, "application/json", bytes.NewReader(body))
	s.NoError(err)
	s.Equal(http.StatusBadRequest, resp.StatusCode)

	var errResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	s.NoError(err)
	s.Contains(errResp, "error")
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
