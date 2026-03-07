-- Drop FK constraint on loinc_code: kb_loinc table is not populated,
-- and LLM OCR cannot reliably produce LOINC codes. Labs use lab_name instead.
ALTER TABLE lab_results DROP CONSTRAINT IF EXISTS lab_results_loinc_code_fkey;

-- Deduplicate medications: prevent duplicate (user, name, dose) rows
CREATE UNIQUE INDEX IF NOT EXISTS idx_medications_user_name_dose
    ON medications (user_id, LOWER(name), LOWER(COALESCE(dose, '')));
