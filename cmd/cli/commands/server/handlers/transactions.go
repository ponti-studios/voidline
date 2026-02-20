package handlers

import (
	"github.com/gin-gonic/gin"
	"voidline/internal/application/services"
)

// TransactionsHandler handles transaction-related requests
type TransactionsHandler struct {
	service *services.FinanceTransactionsService
}

// NewTransactionsHandler creates a new TransactionsHandler
func NewTransactionsHandler(service *services.FinanceTransactionsService) *TransactionsHandler {
	return &TransactionsHandler{service: service}
}

// GetTransactions returns transactions with optional filters
// @Summary Get transactions
// @Description Get financial transactions with optional filters
// @Tags transactions
// @Accept json
// @Produce json
// @Param account query string false "Filter by account"
// @Param category query string false "Filter by category"
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Param page query int false "Page number"
// @Param per_page query int false "Items per page"
// @Success 200 {array} services.FinanceTransactionDTO
// @Router /api/v1/transactions [get]
func (h *TransactionsHandler) GetTransactions(c *gin.Context) {
	opts := services.GetTransactionsOptions{
		Account:   c.Query("account"),
		Category:  c.Query("category"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Page:      1,
		PerPage:   20,
	}

	if p := c.Query("page"); p != "" {
		if page, err := parseInt(p); err == nil {
			opts.Page = page
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if perPage, err := parseInt(pp); err == nil {
			opts.PerPage = perPage
		}
	}

	transactions, err := h.service.GetTransactions(c.Request.Context(), opts)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, transactions)
}

// CreateTransaction creates a new transaction
// @Summary Create a new transaction
// @Description Create a new financial transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Param transaction body services.CreateTransactionInput true "Transaction data"
// @Success 201 {object} services.FinanceTransactionDTO
// @Router /api/v1/transactions [post]
func (h *TransactionsHandler) CreateTransaction(c *gin.Context) {
	var input services.CreateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		HandleError(c, err)
		return
	}

	transaction, err := h.service.CreateTransaction(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	CreatedResponse(c, transaction)
}

// CreateTransactionsBatch creates multiple transactions
// @Summary Create multiple transactions
// @Description Create multiple financial transactions in a batch
// @Tags transactions
// @Accept json
// @Produce json
// @Param batch body services.CreateTransactionsBatchInput true "Batch transaction data"
// @Success 201 {object} services.BatchCreateResult
// @Router /api/v1/transactions/batch [post]
func (h *TransactionsHandler) CreateTransactionsBatch(c *gin.Context) {
	var input services.CreateTransactionsBatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		HandleError(c, err)
		return
	}

	result, err := h.service.CreateTransactionsBatch(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	CreatedResponse(c, result)
}

// GetTransaction returns a single transaction by ID
// @Summary Get transaction by ID
// @Description Get a single financial transaction by ID
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 200 {object} services.FinanceTransactionDTO
// @Router /api/v1/transactions/{id} [get]
func (h *TransactionsHandler) GetTransaction(c *gin.Context) {
	id := c.Param("id")

	transactions, err := h.service.GetTransactions(c.Request.Context(), services.GetTransactionsOptions{})
	if err != nil {
		HandleError(c, err)
		return
	}

	for _, tx := range transactions {
		if tx.ID == id {
			SuccessResponse(c, tx)
			return
		}
	}

	HandleError(c, ErrNotFound("Transaction", id))
}

// UpdateTransaction updates an existing transaction
// @Summary Update a transaction
// @Description Update an existing financial transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param transaction body services.UpdateTransactionInput true "Transaction data"
// @Success 200 {object} services.FinanceTransactionDTO
// @Router /api/v1/transactions/{id} [put]
func (h *TransactionsHandler) UpdateTransaction(c *gin.Context) {
	id := c.Param("id")

	var input services.UpdateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		HandleError(c, err)
		return
	}
	input.ID = id

	transaction, err := h.service.UpdateTransaction(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, transaction)
}

// DeleteTransaction deletes a transaction
// @Summary Delete a transaction
// @Description Delete a financial transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 204
// @Router /api/v1/transactions/{id} [delete]
func (h *TransactionsHandler) DeleteTransaction(c *gin.Context) {
	id := c.Param("id")

	err := h.service.DeleteTransaction(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	NoContentResponse(c)
}

// RegisterRoutes registers transaction routes
func (h *TransactionsHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/transactions", h.GetTransactions)
	r.POST("/transactions", h.CreateTransaction)
	r.POST("/transactions/batch", h.CreateTransactionsBatch)
	r.GET("/transactions/:id", h.GetTransaction)
	r.PUT("/transactions/:id", h.UpdateTransaction)
	r.DELETE("/transactions/:id", h.DeleteTransaction)
}

func parseInt(s string) (int, error) {
	i := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		i = i*10 + int(c-'0')
	}
	return i, nil
}
