package handlers

import (
	"github.com/gin-gonic/gin"
	"voidline/internal/application/services"
)

// DashboardHandler handles dashboard-related requests
type DashboardHandler struct {
	service *services.FinanceDashboardService
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(service *services.FinanceDashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// GetDashboard returns the dashboard summary
// @Summary Get dashboard summary
// @Description Get financial dashboard with total balance, net worth, and accounts
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} services.FinanceDashboardDTO
// @Router /api/v1/dashboard [get]
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	dashboard, err := h.service.GetDashboard(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, dashboard)
}

// RegisterRoutes registers dashboard routes
func (h *DashboardHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/dashboard", h.GetDashboard)
}
