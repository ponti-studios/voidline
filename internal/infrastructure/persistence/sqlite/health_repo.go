package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"voidline/internal/domain/health"
)

type HealthRepository struct {
	db *sql.DB
}

func NewHealthRepository(db *sql.DB) *HealthRepository {
	return &HealthRepository{db: db}
}

func (r *HealthRepository) Create(ctx context.Context, metric *health.HealthMetric) error {
	query := `
		INSERT INTO unified_health_log (timestamp, platform, metric_type, value, unit, source_file)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		metric.Timestamp,
		metric.Platform,
		metric.MetricType,
		metric.Value,
		metric.Unit,
		metric.SourceFile,
	)
	if err != nil {
		return fmt.Errorf("failed to insert health metric: %w", err)
	}
	return nil
}

func (r *HealthRepository) CreateBatch(ctx context.Context, metrics []*health.HealthMetric) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO unified_health_log (timestamp, platform, metric_type, value, unit, source_file)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, metric := range metrics {
		_, err := stmt.ExecContext(ctx,
			metric.Timestamp,
			metric.Platform,
			metric.MetricType,
			metric.Value,
			metric.Unit,
			metric.SourceFile,
		)
		if err != nil {
			return fmt.Errorf("failed to insert health metric: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *HealthRepository) Exists(ctx context.Context, metric *health.HealthMetric) (bool, error) {
	query := `
		SELECT COUNT(*) FROM unified_health_log
		WHERE timestamp = ? AND platform = ? AND metric_type = ? AND source_file = ?
	`
	var count int
	err := r.db.QueryRowContext(ctx, query,
		metric.Timestamp,
		metric.Platform,
		metric.MetricType,
		metric.SourceFile,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check exists: %w", err)
	}
	return count > 0, nil
}

func (r *HealthRepository) InsertWeight(ctx context.Context, w *health.WeightRecord) error {
	query := `INSERT INTO health_weight (timestamp, weight_lb, fat_mass_lb, bone_mass_lb, muscle_mass_lb, hydration_lb, comments, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, w.Timestamp, w.WeightLb, w.FatMassLb, w.BoneMassLb, w.MuscleMassLb, w.HydrationLb, w.Comments, w.Source)
	return err
}

func (r *HealthRepository) InsertSleep(ctx context.Context, s *health.SleepRecord) error {
	query := `INSERT INTO health_sleep (start_time, end_time, light_sleep_seconds, deep_sleep_seconds, rem_sleep_seconds, awake_seconds, wake_up_count, duration_to_sleep_seconds, duration_to_wake_seconds, snoring_seconds, snoring_episodes, avg_heart_rate, min_heart_rate, max_heart_rate, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, s.StartTime, s.EndTime, s.LightSleepSeconds, s.DeepSleepSeconds, s.RemSleepSeconds, s.AwakeSeconds, s.WakeUpCount, s.DurationToSleepSeconds, s.DurationToWakeSeconds, s.SnoringSeconds, s.SnoringEpisodes, s.AvgHeartRate, s.MinHeartRate, s.MaxHeartRate, s.Source)
	return err
}

func (r *HealthRepository) InsertBloodPressure(ctx context.Context, bp *health.BloodPressureRecord) error {
	query := `INSERT INTO health_blood_pressure (timestamp, heart_rate, systolic, diastolic, comments, source) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, bp.Timestamp, bp.HeartRate, bp.Systolic, bp.Diastolic, bp.Comments, bp.Source)
	return err
}

func (r *HealthRepository) InsertHeartRate(ctx context.Context, hr *health.HeartRateRecord) error {
	query := `INSERT INTO health_heart_rate (timestamp, duration_seconds, bpm_value, source) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, hr.Timestamp, hr.DurationSeconds, hr.BpmValue, hr.Source)
	return err
}
