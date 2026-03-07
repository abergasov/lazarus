package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

type DocumentRepo struct {
	db *sqlx.DB
}

func NewDocumentRepo(db *sqlx.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

func (r *DocumentRepo) Create(ctx context.Context, doc *entities.Document) error {
	doc.ID = uuid.New()
	doc.CreatedAt = time.Now()
	if doc.ParseStatus == "" {
		doc.ParseStatus = entities.ParseStatusPending
	}
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO documents (id, user_id, visit_id, storage_key, mime_type, file_name, size_bytes, source_name, source_type, document_date, parse_status, created_at)
		VALUES (:id, :user_id, :visit_id, :storage_key, :mime_type, :file_name, :size_bytes, :source_name, :source_type, :document_date, :parse_status, :created_at)
	`, doc)
	return err
}

func (r *DocumentRepo) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Document, error) {
	var doc entities.Document
	err := r.db.GetContext(ctx, &doc, `SELECT * FROM documents WHERE id = $1 AND user_id = $2`, id, userID)
	return &doc, err
}

// GetInternal fetches a document by ID without user scoping, for internal background processing only.
func (r *DocumentRepo) GetInternal(ctx context.Context, id uuid.UUID) (*entities.Document, error) {
	var doc entities.Document
	err := r.db.GetContext(ctx, &doc, `SELECT * FROM documents WHERE id = $1`, id)
	return &doc, err
}

func (r *DocumentRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]entities.Document, error) {
	var docs []entities.Document
	err := r.db.SelectContext(ctx, &docs, `
		SELECT * FROM documents WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	return docs, err
}

func (r *DocumentRepo) ListPending(ctx context.Context) ([]entities.Document, error) {
	var docs []entities.Document
	err := r.db.SelectContext(ctx, &docs, `
		SELECT * FROM documents WHERE parse_status = $1 ORDER BY created_at ASC
	`, entities.ParseStatusPending)
	return docs, err
}

func (r *DocumentRepo) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM documents WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *DocumentRepo) UpdateMeta(ctx context.Context, id uuid.UUID, category, specialty, summary string, docDate time.Time) {
	_, _ = r.db.ExecContext(ctx, `
		UPDATE documents SET category = $1, specialty = $2, summary = $3, document_date = $4
		WHERE id = $5
	`, category, nilIfEmpty(specialty), nilIfEmpty(summary), docDate, id)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *DocumentRepo) UpdateParseStatus(ctx context.Context, id uuid.UUID, status string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET parse_status = $1, parsed_at = $2 WHERE id = $3`,
		status, now, id)
	return err
}
