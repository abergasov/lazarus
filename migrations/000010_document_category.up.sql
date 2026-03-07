-- Add category column to documents for smart grouping
ALTER TABLE documents ADD COLUMN IF NOT EXISTS category VARCHAR(50) NOT NULL DEFAULT 'other';
ALTER TABLE documents ADD COLUMN IF NOT EXISTS specialty VARCHAR(100);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS summary TEXT;

-- Backfill: documents with lab results linked are "lab_result"
UPDATE documents d SET category = 'lab_result'
WHERE EXISTS (SELECT 1 FROM lab_results lr WHERE lr.document_id = d.id);

CREATE INDEX IF NOT EXISTS idx_documents_category ON documents(user_id, category);
CREATE INDEX IF NOT EXISTS idx_documents_date ON documents(user_id, document_date DESC NULLS LAST);
