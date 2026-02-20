package services

import (
	"database/sql"

	"voidline/internal/domain/account"
	"voidline/internal/domain/category"
	"voidline/internal/domain/transaction"
	"voidline/internal/infrastructure/persistence/sqlite"
)

func newTestTransactionRepository(db *sql.DB) transaction.Repository {
	return sqlite.NewTransactionRepository(db)
}

func newTestAccountRepository(db *sql.DB) account.Repository {
	return sqlite.NewAccountRepository(db)
}

func newTestCategoryRepository(db *sql.DB) category.Repository {
	return sqlite.NewCategoryRepository(db)
}
