-- Prevent duplicate lab results: same user, test name, date, and value
-- This is critical for document re-uploads and re-processing
CREATE UNIQUE INDEX IF NOT EXISTS idx_lab_results_dedup
    ON lab_results (user_id, LOWER(COALESCE(lab_name, '')), collected_at, value);
