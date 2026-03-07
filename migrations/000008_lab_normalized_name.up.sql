-- Add normalized lab name for smarter dedup
ALTER TABLE lab_results ADD COLUMN IF NOT EXISTS normalized_name VARCHAR(255);

-- Backfill: normalize existing lab names
-- Strip parenthetical content, lowercase, trim whitespace
UPDATE lab_results SET normalized_name = LOWER(TRIM(
  REGEXP_REPLACE(COALESCE(lab_name, ''), '\s*\([^)]*\)\s*', ' ')
)) WHERE normalized_name IS NULL;

-- Drop old dedup index and create improved one using normalized_name
DROP INDEX IF EXISTS idx_lab_results_dedup;
CREATE UNIQUE INDEX IF NOT EXISTS idx_lab_results_dedup_v2
    ON lab_results (user_id, LOWER(COALESCE(normalized_name, COALESCE(lab_name, ''))), collected_at, value);
