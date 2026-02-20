package handlers

import (
	"github.com/gin-gonic/gin"
	"voidline/internal/application/services"
)

// CategoriesHandler handles category-related requests
type CategoriesHandler struct {
	service *services.FinanceCategoriesService
}

// NewCategoriesHandler creates a new CategoriesHandler
func NewCategoriesHandler(service *services.FinanceCategoriesService) *CategoriesHandler {
	return &CategoriesHandler{service: service}
}

// GetCategories returns all categories
// @Summary Get all categories
// @Description Get list of all financial categories
// @Tags categories
// @Accept json
// @Produce json
// @Param domain query string false "Filter by domain (finance, health, tracking)"
// @Success 200 {array} services.FinanceCategoryDTO
// @Router /api/v1/categories [get]
func (h *CategoriesHandler) GetCategories(c *gin.Context) {
	domain := c.Query("domain")

	categories, err := h.service.GetCategories(c.Request.Context(), domain)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, categories)
}

// GetCategoryTree returns categories as a hierarchical tree
// @Summary Get category tree
// @Description Get categories as a hierarchical tree structure
// @Tags categories
// @Accept json
// @Produce json
// @Param domain query string false "Filter by domain"
// @Success 200 {array} services.FinanceCategoryDTO
// @Router /api/v1/categories/tree [get]
func (h *CategoriesHandler) GetCategoryTree(c *gin.Context) {
	domain := c.Query("domain")

	categories, err := h.service.GetCategoryTree(c.Request.Context(), domain)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, categories)
}

// CreateCategory creates a new category
// @Summary Create a new category
// @Description Create a new financial category
// @Tags categories
// @Accept json
// @Produce json
// @Param category body services.CreateCategoryInput true "Category data"
// @Success 201 {object} services.FinanceCategoryDTO
// @Router /api/v1/categories [post]
func (h *CategoriesHandler) CreateCategory(c *gin.Context) {
	var input services.CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		HandleError(c, err)
		return
	}

	category, err := h.service.CreateCategory(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	CreatedResponse(c, category)
}

// UpdateCategory updates an existing category
// @Summary Update a category
// @Description Update an existing financial category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param category body services.UpdateCategoryInput true "Category data"
// @Success 200 {object} services.FinanceCategoryDTO
// @Router /api/v1/categories/{id} [put]
func (h *CategoriesHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")

	var input services.UpdateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		HandleError(c, err)
		return
	}
	input.ID = id

	category, err := h.service.UpdateCategory(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, category)
}

// DeleteCategory deletes a category
// @Summary Delete a category
// @Description Delete a financial category and reassign transactions to Uncategorized
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} map[string]int
// @Router /api/v1/categories/{id} [delete]
func (h *CategoriesHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	count, err := h.service.DeleteCategory(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, map[string]int{"reassigned_transactions": count})
}

// RegisterRoutes registers category routes
func (h *CategoriesHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/categories", h.GetCategories)
	r.GET("/categories/tree", h.GetCategoryTree)
	r.POST("/categories", h.CreateCategory)
	r.PUT("/categories/:id", h.UpdateCategory)
	r.DELETE("/categories/:id", h.DeleteCategory)
}
