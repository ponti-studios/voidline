package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"voidline/internal/domain/shared"
	"voidline/internal/domain/tracking"
)

// TrackingEntryRepo implements tracking.TrackingEntryRepository for SQLite
type TrackingEntryRepo struct {
	db *sql.DB
}

// TrackingDataPointRepo implements tracking.TrackingDataPointRepository for SQLite
type TrackingDataPointRepo struct {
	db *sql.DB
}

// TrackingTypeRepo implements tracking.TrackingTypeRepository for SQLite
type TrackingTypeRepo struct {
	db *sql.DB
}

// NewTrackingEntryRepo creates a new SQLite tracking entry repository
func NewTrackingEntryRepo(db *sql.DB) *TrackingEntryRepo {
	return &TrackingEntryRepo{db: db}
}

// NewTrackingDataPointRepo creates a new SQLite tracking data point repository
func NewTrackingDataPointRepo(db *sql.DB) *TrackingDataPointRepo {
	return &TrackingDataPointRepo{db: db}
}

// NewTrackingTypeRepo creates a new SQLite tracking type repository
func NewTrackingTypeRepo(db *sql.DB) *TrackingTypeRepo {
	return &TrackingTypeRepo{db: db}
}

// TrackingEntryRepo implementations

func (r *TrackingEntryRepo) Create(ctx context.Context, entry tracking.TrackingEntry) error {
	metadataJSON, err := entry.GetMetadataJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO tracking_entries (
			id, type, title, content, metadata, date, source_file,
			status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		entry.ID, entry.Type, entry.Title, entry.Content, metadataJSON, entry.Date,
		entry.SourceFile, entry.Status, entry.CreatedAt, entry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create tracking entry: %w", err)
	}

	return nil
}

func (r *TrackingEntryRepo) GetByID(ctx context.Context, id shared.ID) (*tracking.TrackingEntry, error) {
	query := `
		SELECT id, type, title, content, metadata, date, source_file,
			   status, created_at, updated_at
		FROM tracking_entries
		WHERE id = ?
	`

	var entry tracking.TrackingEntry
	var metadataJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID, &entry.Type, &entry.Title, &entry.Content, &metadataJSON, &entry.Date,
		&entry.SourceFile, &entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tracking entry not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tracking entry: %w", err)
	}

	if err := entry.SetMetadataFromJSON(metadataJSON); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &entry, nil
}

func (r *TrackingEntryRepo) Update(ctx context.Context, entry tracking.TrackingEntry) error {
	metadataJSON, err := entry.GetMetadataJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE tracking_entries
		SET type = ?, title = ?, content = ?, metadata = ?, date = ?,
			source_file = ?, status = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		entry.Type, entry.Title, entry.Content, metadataJSON, entry.Date,
		entry.SourceFile, entry.Status, shared.Now(), entry.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tracking entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking entry not found: %s", entry.ID)
	}

	return nil
}

func (r *TrackingEntryRepo) Delete(ctx context.Context, id shared.ID) error {
	query := `DELETE FROM tracking_entries WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tracking entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking entry not found: %s", id)
	}

	return nil
}

func (r *TrackingEntryRepo) List(ctx context.Context, filter tracking.TrackingFilter) ([]tracking.TrackingEntry, error) {
	query := `
		SELECT id, type, title, content, metadata, date, source_file,
			   status, created_at, updated_at
		FROM tracking_entries
		WHERE 1=1
	`

	args := make([]interface{}, 0)

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	if filter.StartDate != nil {
		query += " AND date >= ?"
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != nil {
		query += " AND date <= ?"
		args = append(args, filter.EndDate)
	}

	if filter.SourceFile != "" {
		query += " AND source_file = ?"
		args = append(args, filter.SourceFile)
	}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	query += " ORDER BY date DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracking entries: %w", err)
	}
	defer rows.Close()

	var entries []tracking.TrackingEntry
	for rows.Next() {
		var entry tracking.TrackingEntry
		var metadataJSON string

		err := rows.Scan(
			&entry.ID, &entry.Type, &entry.Title, &entry.Content, &metadataJSON, &entry.Date,
			&entry.SourceFile, &entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking entry: %w", err)
		}

		if err := entry.SetMetadataFromJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *TrackingEntryRepo) Count(ctx context.Context, filter tracking.TrackingFilter) (int, error) {
	query := `SELECT COUNT(*) FROM tracking_entries WHERE 1=1`
	args := make([]interface{}, 0)

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	if filter.StartDate != nil {
		query += " AND date >= ?"
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != nil {
		query += " AND date <= ?"
		args = append(args, filter.EndDate)
	}

	if filter.SourceFile != "" {
		query += " AND source_file = ?"
		args = append(args, filter.SourceFile)
	}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tracking entries: %w", err)
	}

	return count, nil
}

func (r *TrackingEntryRepo) GetByType(ctx context.Context, entryType string, limit int) ([]tracking.TrackingEntry, error) {
	filter := tracking.TrackingFilter{
		Type:  entryType,
		Limit: limit,
	}
	return r.List(ctx, filter)
}

func (r *TrackingEntryRepo) GetByDateRange(ctx context.Context, start, end shared.Timestamp) ([]tracking.TrackingEntry, error) {
	filter := tracking.TrackingFilter{
		StartDate: &start,
		EndDate:   &end,
	}
	return r.List(ctx, filter)
}

func (r *TrackingEntryRepo) GetBySourceFile(ctx context.Context, sourceFile string) ([]tracking.TrackingEntry, error) {
	filter := tracking.TrackingFilter{
		SourceFile: sourceFile,
	}
	return r.List(ctx, filter)
}

func (r *TrackingEntryRepo) Search(ctx context.Context, query string, filter tracking.TrackingFilter) ([]tracking.TrackingEntry, error) {
	searchQuery := `
		SELECT id, type, title, content, metadata, date, source_file,
			   status, created_at, updated_at
		FROM tracking_entries
		WHERE (title LIKE ? OR content LIKE ?)
	`

	args := []interface{}{"%" + query + "%", "%" + query + "%"}

	if filter.Type != "" {
		searchQuery += " AND type = ?"
		args = append(args, filter.Type)
	}

	if filter.StartDate != nil {
		searchQuery += " AND date >= ?"
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != nil {
		searchQuery += " AND date <= ?"
		args = append(args, filter.EndDate)
	}

	searchQuery += " ORDER BY date DESC"

	if filter.Limit > 0 {
		searchQuery += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracking entries: %w", err)
	}
	defer rows.Close()

	var entries []tracking.TrackingEntry
	for rows.Next() {
		var entry tracking.TrackingEntry
		var metadataJSON string

		err := rows.Scan(
			&entry.ID, &entry.Type, &entry.Title, &entry.Content, &metadataJSON, &entry.Date,
			&entry.SourceFile, &entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking entry: %w", err)
		}

		if err := entry.SetMetadataFromJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *TrackingEntryRepo) GetRecentlyUpdated(ctx context.Context, limit int) ([]tracking.TrackingEntry, error) {
	query := `
		SELECT id, type, title, content, metadata, date, source_file,
			   status, created_at, updated_at
		FROM tracking_entries
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recently updated entries: %w", err)
	}
	defer rows.Close()

	var entries []tracking.TrackingEntry
	for rows.Next() {
		var entry tracking.TrackingEntry
		var metadataJSON string

		err := rows.Scan(
			&entry.ID, &entry.Type, &entry.Title, &entry.Content, &metadataJSON, &entry.Date,
			&entry.SourceFile, &entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking entry: %w", err)
		}

		if err := entry.SetMetadataFromJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *TrackingEntryRepo) GetMostActive(ctx context.Context, entryType string, limit int) ([]tracking.TrackingEntry, error) {
	query := `
		SELECT te.id, te.type, te.title, te.content, te.metadata, te.date, te.source_file,
			   te.status, te.created_at, te.updated_at
		FROM tracking_entries te
		LEFT JOIN tracking_data_points tdp ON te.id = tdp.entry_id
		WHERE te.type = ?
		GROUP BY te.id
		ORDER BY COUNT(tdp.id) DESC, te.updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, entryType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get most active entries: %w", err)
	}
	defer rows.Close()

	var entries []tracking.TrackingEntry
	for rows.Next() {
		var entry tracking.TrackingEntry
		var metadataJSON string

		err := rows.Scan(
			&entry.ID, &entry.Type, &entry.Title, &entry.Content, &metadataJSON, &entry.Date,
			&entry.SourceFile, &entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking entry: %w", err)
		}

		if err := entry.SetMetadataFromJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// TrackingDataPointRepo implementations

func (r *TrackingDataPointRepo) Create(ctx context.Context, point tracking.TrackingDataPoint) error {
	metadataJSON := point.Metadata
	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	query := `
		INSERT INTO tracking_data_points (
			id, entry_id, date, type, value, unit, metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		point.ID, point.EntryID, point.Date, point.Type, point.Value, point.Unit,
		metadataJSON, point.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create tracking data point: %w", err)
	}

	return nil
}

func (r *TrackingDataPointRepo) GetByID(ctx context.Context, id shared.ID) (*tracking.TrackingDataPoint, error) {
	query := `
		SELECT id, entry_id, date, type, value, unit, metadata, created_at
		FROM tracking_data_points
		WHERE id = ?
	`

	var point tracking.TrackingDataPoint
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&point.ID, &point.EntryID, &point.Date, &point.Type, &point.Value, &point.Unit,
		&point.Metadata, &point.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tracking data point not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tracking data point: %w", err)
	}

	return &point, nil
}

func (r *TrackingDataPointRepo) Update(ctx context.Context, point tracking.TrackingDataPoint) error {
	query := `
		UPDATE tracking_data_points
		SET date = ?, type = ?, value = ?, unit = ?, metadata = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		point.Date, point.Type, point.Value, point.Unit, point.Metadata, point.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tracking data point: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking data point not found: %s", point.ID)
	}

	return nil
}

func (r *TrackingDataPointRepo) Delete(ctx context.Context, id shared.ID) error {
	query := `DELETE FROM tracking_data_points WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tracking data point: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking data point not found: %s", id)
	}

	return nil
}

func (r *TrackingDataPointRepo) GetByEntryID(ctx context.Context, entryID shared.ID) ([]tracking.TrackingDataPoint, error) {
	query := `
		SELECT id, entry_id, date, type, value, unit, metadata, created_at
		FROM tracking_data_points
		WHERE entry_id = ?
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data points by entry ID: %w", err)
	}
	defer rows.Close()

	var points []tracking.TrackingDataPoint
	for rows.Next() {
		var point tracking.TrackingDataPoint

		err := rows.Scan(
			&point.ID, &point.EntryID, &point.Date, &point.Type, &point.Value, &point.Unit,
			&point.Metadata, &point.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking data point: %w", err)
		}

		points = append(points, point)
	}

	return points, rows.Err()
}

func (r *TrackingDataPointRepo) GetByEntryIDAndType(ctx context.Context, entryID shared.ID, pointType string) ([]tracking.TrackingDataPoint, error) {
	query := `
		SELECT id, entry_id, date, type, value, unit, metadata, created_at
		FROM tracking_data_points
		WHERE entry_id = ? AND type = ?
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, entryID, pointType)
	if err != nil {
		return nil, fmt.Errorf("failed to get data points by entry ID and type: %w", err)
	}
	defer rows.Close()

	var points []tracking.TrackingDataPoint
	for rows.Next() {
		var point tracking.TrackingDataPoint

		err := rows.Scan(
			&point.ID, &point.EntryID, &point.Date, &point.Type, &point.Value, &point.Unit,
			&point.Metadata, &point.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking data point: %w", err)
		}

		points = append(points, point)
	}

	return points, rows.Err()
}

func (r *TrackingDataPointRepo) DeleteByEntryID(ctx context.Context, entryID shared.ID) error {
	query := `DELETE FROM tracking_data_points WHERE entry_id = ?`

	_, err := r.db.ExecContext(ctx, query, entryID)
	if err != nil {
		return fmt.Errorf("failed to delete data points by entry ID: %w", err)
	}

	return nil
}

func (r *TrackingDataPointRepo) GetByTypeAndDateRange(ctx context.Context, pointType string, start, end shared.Timestamp) ([]tracking.TrackingDataPoint, error) {
	query := `
		SELECT id, entry_id, date, type, value, unit, metadata, created_at
		FROM tracking_data_points
		WHERE type = ? AND date >= ? AND date <= ?
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, pointType, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get data points by type and date range: %w", err)
	}
	defer rows.Close()

	var points []tracking.TrackingDataPoint
	for rows.Next() {
		var point tracking.TrackingDataPoint

		err := rows.Scan(
			&point.ID, &point.EntryID, &point.Date, &point.Type, &point.Value, &point.Unit,
			&point.Metadata, &point.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking data point: %w", err)
		}

		points = append(points, point)
	}

	return points, rows.Err()
}

func (r *TrackingDataPointRepo) GetLatestByType(ctx context.Context, pointType string, limit int) ([]tracking.TrackingDataPoint, error) {
	query := `
		SELECT id, entry_id, date, type, value, unit, metadata, created_at
		FROM tracking_data_points
		WHERE type = ?
		ORDER BY date DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, pointType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest data points by type: %w", err)
	}
	defer rows.Close()

	var points []tracking.TrackingDataPoint
	for rows.Next() {
		var point tracking.TrackingDataPoint

		err := rows.Scan(
			&point.ID, &point.EntryID, &point.Date, &point.Type, &point.Value, &point.Unit,
			&point.Metadata, &point.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking data point: %w", err)
		}

		points = append(points, point)
	}

	return points, rows.Err()
}

func (r *TrackingDataPointRepo) GetTimeSeriesData(ctx context.Context, entryID shared.ID, pointType string) ([]tracking.TrackingDataPoint, error) {
	query := `
		SELECT id, entry_id, date, type, value, unit, metadata, created_at
		FROM tracking_data_points
		WHERE entry_id = ? AND type = ?
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, entryID, pointType)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}
	defer rows.Close()

	var points []tracking.TrackingDataPoint
	for rows.Next() {
		var point tracking.TrackingDataPoint

		err := rows.Scan(
			&point.ID, &point.EntryID, &point.Date, &point.Type, &point.Value, &point.Unit,
			&point.Metadata, &point.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking data point: %w", err)
		}

		points = append(points, point)
	}

	return points, rows.Err()
}

// TrackingTypeRepo implementations

func (r *TrackingTypeRepo) Create(ctx context.Context, trackingType tracking.TrackingType) error {
	query := `
		INSERT INTO tracking_types (
			id, name, description, icon, color, schema, is_active,
			status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		trackingType.ID, trackingType.Name, trackingType.Description, trackingType.Icon,
		trackingType.Color, trackingType.Schema, trackingType.IsActive, trackingType.Status,
		trackingType.CreatedAt, trackingType.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create tracking type: %w", err)
	}

	return nil
}

func (r *TrackingTypeRepo) GetByID(ctx context.Context, id shared.ID) (*tracking.TrackingType, error) {
	query := `
		SELECT id, name, description, icon, color, schema, is_active,
			   status, created_at, updated_at
		FROM tracking_types
		WHERE id = ?
	`

	var trackingType tracking.TrackingType
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&trackingType.ID, &trackingType.Name, &trackingType.Description, &trackingType.Icon,
		&trackingType.Color, &trackingType.Schema, &trackingType.IsActive, &trackingType.Status,
		&trackingType.CreatedAt, &trackingType.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tracking type not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tracking type: %w", err)
	}

	return &trackingType, nil
}

func (r *TrackingTypeRepo) GetByName(ctx context.Context, name string) (*tracking.TrackingType, error) {
	query := `
		SELECT id, name, description, icon, color, schema, is_active,
			   status, created_at, updated_at
		FROM tracking_types
		WHERE name = ?
	`

	var trackingType tracking.TrackingType
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&trackingType.ID, &trackingType.Name, &trackingType.Description, &trackingType.Icon,
		&trackingType.Color, &trackingType.Schema, &trackingType.IsActive, &trackingType.Status,
		&trackingType.CreatedAt, &trackingType.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tracking type not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get tracking type by name: %w", err)
	}

	return &trackingType, nil
}

func (r *TrackingTypeRepo) Update(ctx context.Context, trackingType tracking.TrackingType) error {
	query := `
		UPDATE tracking_types
		SET name = ?, description = ?, icon = ?, color = ?, schema = ?,
			is_active = ?, status = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		trackingType.Name, trackingType.Description, trackingType.Icon, trackingType.Color,
		trackingType.Schema, trackingType.IsActive, trackingType.Status, shared.Now(),
		trackingType.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tracking type: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking type not found: %s", trackingType.ID)
	}

	return nil
}

func (r *TrackingTypeRepo) Delete(ctx context.Context, id shared.ID) error {
	query := `DELETE FROM tracking_types WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tracking type: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking type not found: %s", id)
	}

	return nil
}

func (r *TrackingTypeRepo) List(ctx context.Context, activeOnly bool) ([]tracking.TrackingType, error) {
	query := `
		SELECT id, name, description, icon, color, schema, is_active,
			   status, created_at, updated_at
		FROM tracking_types
	`

	args := make([]interface{}, 0)
	if activeOnly {
		query += " WHERE is_active = ?"
		args = append(args, true)
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracking types: %w", err)
	}
	defer rows.Close()

	var types []tracking.TrackingType
	for rows.Next() {
		var trackingType tracking.TrackingType

		err := rows.Scan(
			&trackingType.ID, &trackingType.Name, &trackingType.Description, &trackingType.Icon,
			&trackingType.Color, &trackingType.Schema, &trackingType.IsActive, &trackingType.Status,
			&trackingType.CreatedAt, &trackingType.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracking type: %w", err)
		}

		types = append(types, trackingType)
	}

	return types, rows.Err()
}

func (r *TrackingTypeRepo) GetActive(ctx context.Context) ([]tracking.TrackingType, error) {
	return r.List(ctx, true)
}

func (r *TrackingTypeRepo) SetActive(ctx context.Context, id shared.ID, active bool) error {
	query := `UPDATE tracking_types SET is_active = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, active, shared.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set tracking type active status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking type not found: %s", id)
	}

	return nil
}

func (r *TrackingTypeRepo) UpdateSchema(ctx context.Context, id shared.ID, schema string) error {
	query := `UPDATE tracking_types SET schema = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, schema, shared.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update tracking type schema: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tracking type not found: %s", id)
	}

	return nil
}
