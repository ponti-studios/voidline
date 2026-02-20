package tracking

import (
	"context"

	"voidline/internal/domain/shared"
)

// TrackingEntryRepository handles persistence of tracking entries
type TrackingEntryRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, entry TrackingEntry) error
	GetByID(ctx context.Context, id shared.ID) (*TrackingEntry, error)
	Update(ctx context.Context, entry TrackingEntry) error
	Delete(ctx context.Context, id shared.ID) error

	// Querying operations
	List(ctx context.Context, filter TrackingFilter) ([]TrackingEntry, error)
	Count(ctx context.Context, filter TrackingFilter) (int, error)
	GetByType(ctx context.Context, entryType string, limit int) ([]TrackingEntry, error)
	GetByDateRange(ctx context.Context, start, end shared.Timestamp) ([]TrackingEntry, error)
	GetBySourceFile(ctx context.Context, sourceFile string) ([]TrackingEntry, error)

	// Advanced queries
	Search(ctx context.Context, query string, filter TrackingFilter) ([]TrackingEntry, error)
	GetRecentlyUpdated(ctx context.Context, limit int) ([]TrackingEntry, error)
	GetMostActive(ctx context.Context, entryType string, limit int) ([]TrackingEntry, error)
}

// TrackingDataPointRepository handles persistence of data points
type TrackingDataPointRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, point TrackingDataPoint) error
	GetByID(ctx context.Context, id shared.ID) (*TrackingDataPoint, error)
	Update(ctx context.Context, point TrackingDataPoint) error
	Delete(ctx context.Context, id shared.ID) error

	// Entry-related operations
	GetByEntryID(ctx context.Context, entryID shared.ID) ([]TrackingDataPoint, error)
	GetByEntryIDAndType(ctx context.Context, entryID shared.ID, pointType string) ([]TrackingDataPoint, error)
	DeleteByEntryID(ctx context.Context, entryID shared.ID) error

	// Data analysis operations
	GetByTypeAndDateRange(ctx context.Context, pointType string, start, end shared.Timestamp) ([]TrackingDataPoint, error)
	GetLatestByType(ctx context.Context, pointType string, limit int) ([]TrackingDataPoint, error)
	GetTimeSeriesData(ctx context.Context, entryID shared.ID, pointType string) ([]TrackingDataPoint, error)
}

// TrackingTypeRepository handles persistence of tracking types
type TrackingTypeRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, trackingType TrackingType) error
	GetByID(ctx context.Context, id shared.ID) (*TrackingType, error)
	GetByName(ctx context.Context, name string) (*TrackingType, error)
	Update(ctx context.Context, trackingType TrackingType) error
	Delete(ctx context.Context, id shared.ID) error

	// Listing operations
	List(ctx context.Context, activeOnly bool) ([]TrackingType, error)
	GetActive(ctx context.Context) ([]TrackingType, error)

	// Management operations
	SetActive(ctx context.Context, id shared.ID, active bool) error
	UpdateSchema(ctx context.Context, id shared.ID, schema string) error
}

// TrackingService combines all repositories for business logic
type TrackingService struct {
	EntryRepo     TrackingEntryRepository
	DataPointRepo TrackingDataPointRepository
	TypeRepo      TrackingTypeRepository
	EntityRepo    shared.EntityRepository
	SearchRepo    shared.SearchRepository
}

// NewTrackingService creates a new tracking service with all dependencies
func NewTrackingService(
	entryRepo TrackingEntryRepository,
	dataPointRepo TrackingDataPointRepository,
	typeRepo TrackingTypeRepository,
	entityRepo shared.EntityRepository,
	searchRepo shared.SearchRepository,
) *TrackingService {
	return &TrackingService{
		EntryRepo:     entryRepo,
		DataPointRepo: dataPointRepo,
		TypeRepo:      typeRepo,
		EntityRepo:    entityRepo,
		SearchRepo:    searchRepo,
	}
}

// CreateEntryWithDataPoints creates an entry and all its data points in a transaction
func (s *TrackingService) CreateEntryWithDataPoints(ctx context.Context, entry TrackingEntry) error {
	// Create the main entry
	if err := s.EntryRepo.Create(ctx, entry); err != nil {
		return err
	}

	// Create corresponding universal entity for search
	entityMetadata, _ := entry.GetMetadataJSON()
	baseEntity := shared.BaseEntity{
		ID:        entry.ID,
		Domain:    "tracking",
		Type:      entry.Type,
		Title:     entry.Title,
		Status:    entry.Status,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}

	if err := s.EntityRepo.Create(ctx, baseEntity); err != nil {
		// Log but don't fail - entity creation is for search indexing
	}

	// Index for search
	content := entry.Content + " " + entityMetadata
	if err := s.SearchRepo.Index(ctx, &entry, content); err != nil {
		// Log but don't fail - search indexing is optional
	}

	// Create data points
	for _, point := range entry.DataPoints {
		point.EntryID = entry.ID
		if err := s.DataPointRepo.Create(ctx, point); err != nil {
			return err
		}
	}

	return nil
}

// GetEntryWithDataPoints retrieves an entry with all its data points
func (s *TrackingService) GetEntryWithDataPoints(ctx context.Context, id shared.ID) (*TrackingEntry, error) {
	entry, err := s.EntryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	dataPoints, err := s.DataPointRepo.GetByEntryID(ctx, id)
	if err != nil {
		return entry, nil // Return entry even if data points fail
	}

	entry.DataPoints = dataPoints
	return entry, nil
}

// DeleteEntryWithDataPoints removes an entry and all its data points
func (s *TrackingService) DeleteEntryWithDataPoints(ctx context.Context, id shared.ID) error {
	// Delete data points first
	if err := s.DataPointRepo.DeleteByEntryID(ctx, id); err != nil {
		return err
	}

	// Remove from search index
	s.SearchRepo.RemoveFromIndex(ctx, id)

	// Delete from universal entities
	s.EntityRepo.Delete(ctx, id)

	// Delete the entry
	return s.EntryRepo.Delete(ctx, id)
}
