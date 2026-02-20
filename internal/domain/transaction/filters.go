package transaction

import (
	"voidline/internal/domain/timeutil"
)

// Filter represents the filter criteria for querying transactions
type Filter struct {
	StartDate        *timeutil.Date
	EndDate          *timeutil.Date
	Accounts         []string
	Categories       []string
	ParentCategories []string
	MinAmount        *float64
	MaxAmount        *float64
	SearchQuery      string
	Recurring        *bool
	Excluded         *bool
	Tags             []string
	TransactionType  string
	Page             int
	PerPage          int
}

// DefaultPerPage is the default number of items per page
const DefaultPerPage = 20

// NewFilter creates a new Filter with defaults
func NewFilter() Filter {
	return Filter{
		Page:    1,
		PerPage: DefaultPerPage,
	}
}

// WithPage sets the page number
func (f Filter) WithPage(page int) Filter {
	f.Page = page
	return f
}

// WithPerPage sets the items per page
func (f Filter) WithPerPage(perPage int) Filter {
	f.PerPage = perPage
	return f
}

// WithDateRange sets the date range filter
func (f Filter) WithDateRange(start, end timeutil.Date) Filter {
	f.StartDate = &start
	f.EndDate = &end
	return f
}

// WithAccounts sets the accounts filter
func (f Filter) WithAccounts(accounts ...string) Filter {
	f.Accounts = accounts
	return f
}

// WithCategories sets the categories filter
func (f Filter) WithCategories(categories ...string) Filter {
	f.Categories = categories
	return f
}

// WithAmountRange sets the amount range filter
func (f Filter) WithAmountRange(min, max float64) Filter {
	f.MinAmount = &min
	f.MaxAmount = &max
	return f
}

// WithSearch sets the search query filter
func (f Filter) WithSearch(query string) Filter {
	f.SearchQuery = query
	return f
}

// WithRecurring sets the recurring filter
func (f Filter) WithRecurring(recurring bool) Filter {
	f.Recurring = &recurring
	return f
}

// WithExcluded sets the excluded filter
func (f Filter) WithExcluded(excluded bool) Filter {
	f.Excluded = &excluded
	return f
}

// IsEmpty returns true if no filters are applied
func (f Filter) IsEmpty() bool {
	return f.StartDate == nil && f.EndDate == nil &&
		len(f.Accounts) == 0 && len(f.Categories) == 0 &&
		f.MinAmount == nil && f.MaxAmount == nil &&
		f.SearchQuery == "" && f.Recurring == nil && f.Excluded == nil
}

// PaginatedResult represents a paginated list of transactions
type PaginatedResult struct {
	Items      []Transaction
	TotalCount int
	Page       int
	PerPage    int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

// NewPaginatedResult creates a new PaginatedResult
func NewPaginatedResult(items []Transaction, totalCount, page, perPage int) PaginatedResult {
	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginatedResult{
		Items:      items,
		TotalCount: totalCount,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// Offset returns the database offset for the current page
func (p PaginatedResult) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit returns the database limit
func (p PaginatedResult) Limit() int {
	return p.PerPage
}
