package tracking

import (
	"encoding/json"
	"time"

	"voidline/internal/domain/shared"
)

// TrackingEntry represents a tracking entry from markdown files or new entries
type TrackingEntry struct {
	shared.BaseEntity
	Type       string                 `json:"type" db:"type"`               // health, career, activity, etc.
	Content    string                 `json:"content" db:"content"`         // original markdown content
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`       // YAML frontmatter data
	Date       shared.Timestamp       `json:"date" db:"date"`               // primary date for the entry
	SourceFile string                 `json:"source_file" db:"source_file"` // original file path if imported
	DataPoints []TrackingDataPoint    `json:"data_points,omitempty"`        // associated data points
}

// TrackingDataPoint represents individual data points within a tracking entry
// For example, individual medication doses, hike dates, book entries, etc.
type TrackingDataPoint struct {
	ID        shared.ID        `json:"id" db:"id"`
	EntryID   shared.ID        `json:"entry_id" db:"entry_id"`
	Date      shared.Timestamp `json:"date" db:"date"`
	Type      string           `json:"type" db:"type"`         // measurement, event, goal, medication, etc.
	Value     string           `json:"value" db:"value"`       // flexible value field
	Unit      string           `json:"unit" db:"unit"`         // mg, hours, count, percentage, etc.
	Metadata  string           `json:"metadata" db:"metadata"` // JSON for additional structured data
	CreatedAt shared.Timestamp `json:"created_at" db:"created_at"`
}

// TrackingType represents different types of tracking (health, career, etc.)
type TrackingType struct {
	shared.BaseEntity
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	Icon        string `json:"icon" db:"icon"`     // emoji icon
	Color       string `json:"color" db:"color"`   // hex color
	Schema      string `json:"schema" db:"schema"` // JSON schema for data points
	IsActive    bool   `json:"is_active" db:"is_active"`
}

// NewTrackingEntry creates a new tracking entry
func NewTrackingEntry(entryType, title, content string, date shared.Timestamp) TrackingEntry {
	base := shared.NewBaseEntity("tracking", "entry", title)
	return TrackingEntry{
		BaseEntity: base,
		Type:       entryType,
		Content:    content,
		Metadata:   make(map[string]interface{}),
		Date:       date,
	}
}

// NewTrackingDataPoint creates a new tracking data point
func NewTrackingDataPoint(entryID shared.ID, pointType, value, unit string, date shared.Timestamp) TrackingDataPoint {
	return TrackingDataPoint{
		ID:        shared.NewID("tracking_data"),
		EntryID:   entryID,
		Date:      date,
		Type:      pointType,
		Value:     value,
		Unit:      unit,
		CreatedAt: shared.Now(),
	}
}

// NewTrackingType creates a new tracking type
func NewTrackingType(name, description, icon, color string) TrackingType {
	base := shared.NewBaseEntity("tracking", "type", name)
	return TrackingType{
		BaseEntity:  base,
		Name:        name,
		Description: description,
		Icon:        icon,
		Color:       color,
		IsActive:    true,
	}
}

// GetMetadataJSON returns the metadata as a JSON string for database storage
func (te *TrackingEntry) GetMetadataJSON() (string, error) {
	if te.Metadata == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(te.Metadata)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// SetMetadataFromJSON parses JSON string into metadata map
func (te *TrackingEntry) SetMetadataFromJSON(jsonStr string) error {
	if jsonStr == "" {
		te.Metadata = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &te.Metadata)
}

// AddDataPoint adds a data point to the entry
func (te *TrackingEntry) AddDataPoint(point TrackingDataPoint) {
	if te.DataPoints == nil {
		te.DataPoints = make([]TrackingDataPoint, 0)
	}
	te.DataPoints = append(te.DataPoints, point)
}

// GetDataPointsByType returns all data points of a specific type
func (te *TrackingEntry) GetDataPointsByType(pointType string) []TrackingDataPoint {
	var results []TrackingDataPoint
	for _, point := range te.DataPoints {
		if point.Type == pointType {
			results = append(results, point)
		}
	}
	return results
}

// GetDateRange returns the earliest and latest dates from the entry and its data points
func (te *TrackingEntry) GetDateRange() (earliest, latest time.Time) {
	earliest = te.Date.Time()
	latest = te.Date.Time()

	for _, point := range te.DataPoints {
		pointTime := point.Date.Time()
		if pointTime.Before(earliest) {
			earliest = pointTime
		}
		if pointTime.After(latest) {
			latest = pointTime
		}
	}

	return earliest, latest
}

// Common tracking types
var (
	DefaultTrackingTypes = []TrackingType{
		{
			BaseEntity:  shared.NewBaseEntity("tracking", "type", "Health"),
			Name:        "Health",
			Description: "Medical, fitness, and wellness tracking",
			Icon:        "🏥",
			Color:       "#E53E3E",
			IsActive:    true,
		},
		{
			BaseEntity:  shared.NewBaseEntity("tracking", "type", "Career"),
			Name:        "Career",
			Description: "Job applications, interviews, and career progression",
			Icon:        "💼",
			Color:       "#3182CE",
			IsActive:    true,
		},
		{
			BaseEntity:  shared.NewBaseEntity("tracking", "type", "Activity"),
			Name:        "Activity",
			Description: "Hobbies, entertainment, and recreational activities",
			Icon:        "🎯",
			Color:       "#38A169",
			IsActive:    true,
		},
		{
			BaseEntity:  shared.NewBaseEntity("tracking", "type", "Habit"),
			Name:        "Habit",
			Description: "Daily habits and routine tracking",
			Icon:        "📈",
			Color:       "#805AD5",
			IsActive:    true,
		},
		{
			BaseEntity:  shared.NewBaseEntity("tracking", "type", "Goal"),
			Name:        "Goal",
			Description: "Personal goals and progress tracking",
			Icon:        "🎯",
			Color:       "#D69E2E",
			IsActive:    true,
		},
	}
)

// TrackingFilter for querying tracking entries
type TrackingFilter struct {
	Type       string
	StartDate  *shared.Timestamp
	EndDate    *shared.Timestamp
	Tags       []string
	Status     string
	SourceFile string
	Limit      int
	Offset     int
}

// DefaultTrackingFilter returns a filter with sensible defaults
func DefaultTrackingFilter() TrackingFilter {
	return TrackingFilter{
		Limit:  20,
		Offset: 0,
		Status: "active",
	}
}
