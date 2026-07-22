-- +goose Up
ALTER TABLE places ADD COLUMN review_status TEXT;
ALTER TABLE places ADD COLUMN review_reason TEXT;
ALTER TABLE places ADD COLUMN review_query TEXT;
ALTER TABLE places ADD COLUMN review_updated_at TEXT;
ALTER TABLE places ADD COLUMN review_decision_at TEXT;
ALTER TABLE places ADD COLUMN review_decision_source TEXT;
ALTER TABLE places ADD COLUMN last_geocode_status TEXT;
ALTER TABLE places ADD COLUMN last_geocode_query TEXT;
ALTER TABLE places ADD COLUMN last_geocode_result_summary TEXT;

-- +goose Down
-- These columns are additive and read by geokit-review alongside the
-- normalized place_review_state and place_geocode_state tables.
-- Down migration is intentionally empty — columns are non-destructive.
