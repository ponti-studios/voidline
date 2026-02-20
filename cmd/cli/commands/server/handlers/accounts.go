package handlers

import (
	"github.com/gin-gonic/gin"
	"voidline/internal/application/services"
)

// AccountsHandler handles account-related requests
type AccountsHandler struct {
	service *services.FinanceAccountsService
}

// NewAccountsHandler creates a new AccountsHandler
func NewAccountsHandler(service *services.FinanceAccountsService) *AccountsHandler {
	return &AccountsHandler{service: service}
}

// GetAccounts returns all accounts
// @Summary Get all accounts
// @Description Get list of all financial accounts with balances
// @Tags accounts
// @Accept json
// @Produce json
// @Success 200 {array} services.FinanceAccountDTO
// @Router /api/v1/accounts [get]
func (h *AccountsHandler) GetAccounts(c *gin.Context) {
	accounts, err := h.service.GetAccounts(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, accounts)
}

// CreateAccount creates a new account
// @Summary Create a new account
// @Description Create a new financial account
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body services.CreateAccountInput true "Account data"
// @Success 201 {object} services.FinanceAccountDTO
// @Router /api/v1/accounts [post]
func (h *AccountsHandler) CreateAccount(c *gin.Context) {
	var input services.CreateAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		HandleError(c, err)
		return
	}

	account, err := h.service.CreateAccount(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	CreatedResponse(c, account)
}

// GetAccount returns a single account by ID
// @Summary Get account by ID
// @Description Get a single financial account by ID
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} services.FinanceAccountDTO
// @Router /api/v1/accounts/{id} [get]
func (h *AccountsHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")

	accounts, err := h.service.GetAccounts(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	for _, acc := range accounts {
		if acc.ID == id {
			SuccessResponse(c, acc)
			return
		}
	}

	HandleError(c, ErrNotFound("Account", id))
}

// UpdateAccount updates an existing account
// @Summary Update an account
// @Description Update an existing financial account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param account body services.UpdateAccountInput true "Account data"
// @Success 200 {object} services.FinanceAccountDTO
// @Router /api/v1/accounts/{id} [put]
func (h *AccountsHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")

	var input services.UpdateAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		HandleError(c, err)
		return
	}
	input.ID = id

	account, err := h.service.UpdateAccount(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, account)
}

// DeleteAccount deletes an account
// @Summary Delete an account
// @Description Delete a financial account and all its transactions
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} map[string]int
// @Router /api/v1/accounts/{id} [delete]
func (h *AccountsHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")

	count, err := h.service.DeleteAccount(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, map[string]int{"deleted_transactions": count})
}

// RegisterRoutes registers account routes
func (h *AccountsHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/accounts", h.GetAccounts)
	r.POST("/accounts", h.CreateAccount)
	r.GET("/accounts/:id", h.GetAccount)
	r.PUT("/accounts/:id", h.UpdateAccount)
	r.DELETE("/accounts/:id", h.DeleteAccount)
}

func ErrNotFound(resource, id string) error {
	return &notFoundError{resource: resource, id: id}
}

type notFoundError struct {
	resource string
	id       string
}

func (e *notFoundError) Error() string {
	return e.resource + " not found: " + e.id
}
