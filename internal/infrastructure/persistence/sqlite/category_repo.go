package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"voidline/internal/domain/category"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) FindByID(ctx context.Context, id int) (*category.Category, error) {
	query := `
		SELECT id, name, parent_id, domain, description, is_active, created_at, updated_at
		FROM categories
		WHERE id = ?
	`

	var cat category.Category
	var parentID sql.NullInt64
	var description sql.NullString
	var isActive int
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cat.ID, &cat.Name, &parentID, &cat.Domain, &description, &isActive,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	if parentID.Valid {
		cat.ParentID = new(int)
		*cat.ParentID = int(parentID.Int64)
	}
	cat.Description = description.String
	cat.IsActive = isActive == 1

	if createdAt != "" {
		cat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	if updatedAt != "" {
		cat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	}

	return &cat, nil
}

func (r *CategoryRepository) FindByName(ctx context.Context, name string, domain category.Domain) (*category.Category, error) {
	query := `
		SELECT id, name, parent_id, domain, description, is_active, created_at, updated_at
		FROM categories
		WHERE name = ? AND domain = ?
	`

	var cat category.Category
	var parentID sql.NullInt64
	var description sql.NullString
	var isActive int
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, query, name, domain).Scan(
		&cat.ID, &cat.Name, &parentID, &cat.Domain, &description, &isActive,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find category by name: %w", err)
	}

	cat.Description = description.String
	cat.IsActive = isActive == 1
	if parentID.Valid {
		cat.ParentID = new(int)
		*cat.ParentID = int(parentID.Int64)
	}
	if createdAt != "" {
		cat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	if updatedAt != "" {
		cat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	}

	return &cat, nil
}

func scanCategory(rows *sql.Rows) (*category.Category, error) {
	var cat category.Category
	var parentID sql.NullInt64
	var description sql.NullString
	var isActive int
	var createdAt, updatedAt string

	err := rows.Scan(
		&cat.ID, &cat.Name, &parentID, &cat.Domain, &description, &isActive,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	cat.Description = description.String
	cat.IsActive = isActive == 1
	if parentID.Valid {
		cat.ParentID = new(int)
		*cat.ParentID = int(parentID.Int64)
	}
	if createdAt != "" {
		cat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	if updatedAt != "" {
		cat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	}

	return &cat, nil
}

func (r *CategoryRepository) FindByDomain(ctx context.Context, domain category.Domain) ([]category.Category, error) {
	query := `
		SELECT id, name, parent_id, domain, description, is_active, created_at, updated_at
		FROM categories
		WHERE domain = ? AND is_active = 1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to find categories by domain: %w", err)
	}
	defer rows.Close()

	var categories []category.Category
	for rows.Next() {
		cat, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, *cat)
	}

	return categories, nil
}

func (r *CategoryRepository) FindChildren(ctx context.Context, parentID int) ([]category.Category, error) {
	query := `
		SELECT id, name, parent_id, domain, description, is_active, created_at, updated_at
		FROM categories
		WHERE parent_id = ? AND is_active = 1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find child categories: %w", err)
	}
	defer rows.Close()

	var categories []category.Category
	for rows.Next() {
		cat, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, *cat)
	}

	return categories, nil
}

func (r *CategoryRepository) FindAll(ctx context.Context) ([]category.Category, error) {
	query := `
		SELECT id, name, parent_id, domain, description, is_active, created_at, updated_at
		FROM categories
		WHERE is_active = 1
		ORDER BY domain, name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find all categories: %w", err)
	}
	defer rows.Close()

	var categories []category.Category
	for rows.Next() {
		cat, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, *cat)
	}

	return categories, nil
}

func (r *CategoryRepository) Create(ctx context.Context, cat *category.Category) error {
	query := `
		INSERT INTO categories (name, parent_id, domain, description, is_active)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		cat.Name, cat.ParentID, cat.Domain, cat.Description, cat.IsActive,
	)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get insert id: %w", err)
	}

	cat.ID = int(id)
	return nil
}

func (r *CategoryRepository) Update(ctx context.Context, cat *category.Category) error {
	query := `
		UPDATE categories
		SET name = ?, parent_id = ?, domain = ?, description = ?, is_active = ?, updated_at = datetime('now')
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		cat.Name, cat.ParentID, cat.Domain, cat.Description, cat.IsActive, cat.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM categories WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) HasChildren(ctx context.Context, id int) (bool, error) {
	query := `SELECT COUNT(*) FROM categories WHERE parent_id = ? AND is_active = 1`

	var count int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check for children: %w", err)
	}

	return count > 0, nil
}

func (r *CategoryRepository) GetTree(ctx context.Context, domain category.Domain) ([]category.Category, error) {
	allCategories, err := r.FindByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}

	var roots []category.Category
	categoryMap := make(map[int]*category.Category)

	// Build map of all categories
	for i := range allCategories {
		cat := &allCategories[i]
		cat.Children = make([]*category.Category, 0)
		categoryMap[cat.ID] = cat
	}

	// Build tree structure by assigning children to parents
	for i := range allCategories {
		cat := &allCategories[i]
		if cat.ParentID == nil {
			roots = append(roots, *cat)
		} else {
			parent, exists := categoryMap[*cat.ParentID]
			if exists {
				parent.AddChild(cat)
			}
		}
	}

	return roots, nil
}

func (r *CategoryRepository) CountByCategory(ctx context.Context, categoryName string, domain category.Domain) (int, error) {
	query := `SELECT COUNT(*) FROM finance_transactions WHERE category = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, categoryName).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions by category: %w", err)
	}

	return count, nil
}

func (r *CategoryRepository) ReassignTransactions(ctx context.Context, fromName, toName string, domain category.Domain) (int, error) {
	query := `UPDATE finance_transactions SET category = ? WHERE category = ?`

	result, err := r.db.ExecContext(ctx, query, toName, fromName)
	if err != nil {
		return 0, fmt.Errorf("failed to reassign transactions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}
