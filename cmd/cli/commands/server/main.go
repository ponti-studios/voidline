package server

import (
	"fmt"
	"os"

	"gogogo/cmd/cli/commands/server/handlers"
	"gogogo/internal/application/services"
	"gogogo/internal/infrastructure/config"
	"gogogo/internal/infrastructure/persistence/sqlite"

	_ "gogogo/docs"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Finance API
// @version 1.0
// @description REST API for financial transaction management
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @schemes http
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	db, err := sqlite.NewConnection(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

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

	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		accountsHandler.RegisterRoutes(api)
		transactionsHandler.RegisterRoutes(api)
		categoriesHandler.RegisterRoutes(api)
		dashboardHandler.RegisterRoutes(api)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "gogogo",
	})
	logger.Info("starting REST API server", "url", fmt.Sprintf("http://localhost:%s", port))
	logger.Info("swagger UI available", "url", fmt.Sprintf("http://localhost:%s/swagger/index.html", port))

	return r.Run(":" + port)
}
