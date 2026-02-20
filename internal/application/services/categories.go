package services

import (
	"context"
	"fmt"

	"voidline/internal/application/errors"
	"voidline/internal/application/validation"
	"voidline/internal/domain/category"
)

type FinanceCategoriesService struct {
	categoryRepo category.Repository
}

func NewFinanceCategoriesService(categoryRepo category.Repository) *FinanceCategoriesService {
	return &FinanceCategoriesService{categoryRepo: categoryRepo}
}

type FinanceCategoryDTO struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	ParentID    *string              `json:"parentId"`
	Domain      string               `json:"domain"`
	Description string               `json:"description"`
	Children    []FinanceCategoryDTO `json:"children,omitempty"`
}

type CreateCategoryInput struct {
	Name        string  `json:"name" validate:"required"`
	ParentID    *string `json:"parentId"`
	Domain      string  `json:"domain" validate:"required"`
	Description string  `json:"description"`
}

type UpdateCategoryInput struct {
	ID          string  `json:"id" validate:"required"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parentId"`
	Description string  `json:"description"`
}

func (s *FinanceCategoriesService) GetCategories(ctx context.Context, domain string) ([]FinanceCategoryDTO, error) {
	var categories []category.Category
	var err error

	if domain != "" {
		categories, err = s.categoryRepo.FindByDomain(ctx, category.Domain(domain))
	} else {
		categories, err = s.categoryRepo.FindAll(ctx)
	}
	if err != nil {
		return nil, err
	}

	dtos := make([]FinanceCategoryDTO, len(categories))
	for i, cat := range categories {
		dtos[i] = categoryToDTO(cat)
	}

	return dtos, nil
}

func (s *FinanceCategoriesService) GetCategoryTree(ctx context.Context, domain string) ([]FinanceCategoryDTO, error) {
	var categories []category.Category
	var err error

	if domain != "" {
		categories, err = s.categoryRepo.FindByDomain(ctx, category.Domain(domain))
	} else {
		categories, err = s.categoryRepo.FindAll(ctx)
	}
	if err != nil {
		return nil, err
	}

	return buildCategoryTree(categories), nil
}

func (s *FinanceCategoriesService) CreateCategory(ctx context.Context, input CreateCategoryInput) (FinanceCategoryDTO, error) {
	var errs validation.ValidationErrors

	if err := validation.ValidateRequired("name", input.Name); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateRequired("domain", input.Domain); err != nil {
		errs = append(errs, *err)
	}
	if err := validation.ValidateDomain("domain", input.Domain); err != nil {
		errs = append(errs, *err)
	}

	if errs.HasErrors() {
		return FinanceCategoryDTO{}, errs
	}

	domain := category.Domain(input.Domain)

	if input.ParentID != nil && *input.ParentID != "" {
		var parentID int
		fmt.Sscanf(*input.ParentID, "%d", &parentID)

		parent, err := s.categoryRepo.FindByID(ctx, parentID)
		if err != nil {
			return FinanceCategoryDTO{}, fmt.Errorf("failed to find parent category: %w", err)
		}
		if parent == nil {
			return FinanceCategoryDTO{}, errors.NotFoundError{Resource: "Parent category", ID: *input.ParentID}
		}
	}

	existing, err := s.categoryRepo.FindByName(ctx, input.Name, domain)
	if err != nil {
		return FinanceCategoryDTO{}, fmt.Errorf("failed to check existing category: %w", err)
	}
	if existing != nil {
		return FinanceCategoryDTO{}, fmt.Errorf("category already exists: %s", input.Name)
	}

	var parentID *int
	if input.ParentID != nil && *input.ParentID != "" {
		var pid int
		fmt.Sscanf(*input.ParentID, "%d", &pid)
		parentID = &pid
	}

	cat := &category.Category{
		Name:        input.Name,
		ParentID:    parentID,
		Domain:      domain,
		Description: input.Description,
		IsActive:    true,
	}

	err = s.categoryRepo.Create(ctx, cat)
	if err != nil {
		return FinanceCategoryDTO{}, fmt.Errorf("failed to create category: %w", err)
	}

	return categoryToDTO(*cat), nil
}

func (s *FinanceCategoriesService) UpdateCategory(ctx context.Context, input UpdateCategoryInput) (FinanceCategoryDTO, error) {
	var errs validation.ValidationErrors

	if err := validation.ValidateRequired("id", input.ID); err != nil {
		errs = append(errs, *err)
	}

	if errs.HasErrors() {
		return FinanceCategoryDTO{}, errs
	}

	var id int
	fmt.Sscanf(input.ID, "%d", &id)

	existing, err := s.categoryRepo.FindByID(ctx, id)
	if err != nil {
		return FinanceCategoryDTO{}, fmt.Errorf("failed to find category: %w", err)
	}
	if existing == nil {
		return FinanceCategoryDTO{}, errors.NotFoundError{Resource: "Category", ID: input.ID}
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.ParentID != nil {
		if *input.ParentID == "" {
			existing.ParentID = nil
		} else {
			var parentID int
			fmt.Sscanf(*input.ParentID, "%d", &parentID)
			if parentID == existing.ID {
				return FinanceCategoryDTO{}, fmt.Errorf("category cannot be its own parent")
			}
			existing.ParentID = &parentID
		}
	}
	if input.Description != "" {
		existing.Description = input.Description
	}

	err = s.categoryRepo.Update(ctx, existing)
	if err != nil {
		return FinanceCategoryDTO{}, fmt.Errorf("failed to update category: %w", err)
	}

	return categoryToDTO(*existing), nil
}

func (s *FinanceCategoriesService) DeleteCategory(ctx context.Context, id string) (int, error) {
	if err := validation.ValidateRequired("id", id); err != nil {
		return 0, err
	}

	var catID int
	fmt.Sscanf(id, "%d", &catID)

	existing, err := s.categoryRepo.FindByID(ctx, catID)
	if err != nil {
		return 0, fmt.Errorf("failed to find category: %w", err)
	}
	if existing == nil {
		return 0, errors.NotFoundError{Resource: "Category", ID: id}
	}

	if existing.Name == category.UncategorizedName {
		return 0, errors.CannotDeleteError{Resource: "Category", Reason: "cannot delete Uncategorized category"}
	}

	hasChildren, err := s.categoryRepo.HasChildren(ctx, catID)
	if err != nil {
		return 0, fmt.Errorf("failed to check for children: %w", err)
	}
	if hasChildren {
		return 0, fmt.Errorf("cannot delete category with children. Delete or reassign children first")
	}

	count, err := s.categoryRepo.CountByCategory(ctx, existing.Name, existing.Domain)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	if count > 0 {
		_, err = s.categoryRepo.ReassignTransactions(ctx, existing.Name, category.UncategorizedName, existing.Domain)
		if err != nil {
			return 0, errors.ReassignError{From: existing.Name, To: category.UncategorizedName, Count: count}
		}
	}

	err = s.categoryRepo.Delete(ctx, catID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete category: %w", err)
	}

	return count, nil
}

func categoryToDTO(cat category.Category) FinanceCategoryDTO {
	dto := FinanceCategoryDTO{
		ID:          fmt.Sprintf("%d", cat.ID),
		Name:        cat.Name,
		Domain:      string(cat.Domain),
		Description: cat.Description,
	}

	if cat.ParentID != nil {
		parentID := fmt.Sprintf("%d", *cat.ParentID)
		dto.ParentID = &parentID
	}

	return dto
}

func buildCategoryTree(categories []category.Category) []FinanceCategoryDTO {
	categoryMap := make(map[int]*category.Category)
	for i := range categories {
		categoryMap[categories[i].ID] = &categories[i]
	}

	var roots []FinanceCategoryDTO
	for i := range categories {
		cat := &categories[i]
		dto := categoryToDTO(*cat)

		if cat.ParentID == nil {
			roots = append(roots, dto)
		} else {
			if _, exists := categoryMap[*cat.ParentID]; exists {
				parentDTO := findFinanceCategoryDTOByID(roots, *cat.ParentID)
				if parentDTO != nil {
					parentDTO.Children = append(parentDTO.Children, dto)
				}
			}
		}
	}

	return roots
}

func findFinanceCategoryDTOByID(categories []FinanceCategoryDTO, id int) *FinanceCategoryDTO {
	for i := range categories {
		var catID int
		fmt.Sscanf(categories[i].ID, "%d", &catID)
		if catID == id {
			return &categories[i]
		}
		if len(categories[i].Children) > 0 {
			if found := findFinanceCategoryDTOByID(categories[i].Children, id); found != nil {
				return found
			}
		}
	}
	return nil
}
