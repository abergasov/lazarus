package document

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
	"lazarus/internal/repository"
	"lazarus/internal/storage/bucket"
)

type Service struct {
	db          *sqlx.DB
	docRepo     *repository.DocumentRepo
	bucket      *bucket.Client
	providerReg *provider.Registry
}

func NewService(db *sqlx.DB, s3 *bucket.Client, providerReg *provider.Registry) *Service {
	return &Service{
		db:          db,
		docRepo:     repository.NewDocumentRepo(db),
		bucket:      s3,
		providerReg: providerReg,
	}
}

func (s *Service) Upload(ctx context.Context, userID uuid.UUID, visitIDStr string, file *multipart.FileHeader, sourceType string) (*entities.Document, error) {
	f, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open upload: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}

	ext := filepath.Ext(file.Filename)
	storageKey := fmt.Sprintf("documents/%s/%s%s", userID, uuid.New(), ext)
	contentType := file.Header.Get("Content-Type")

	if err := s.bucket.Upload(ctx, storageKey, bytes.NewReader(data), int64(len(data))); err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}
	_ = contentType

	doc := &entities.Document{
		UserID:     userID,
		StorageKey: storageKey,
		MimeType:   contentType,
		FileName:   file.Filename,
		SizeBytes:  file.Size,
		SourceType: sourceType,
	}
	if visitIDStr != "" {
		vid, err := uuid.Parse(visitIDStr)
		if err == nil {
			doc.VisitID = &vid
		}
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("save document: %w", err)
	}
	return doc, nil
}

func (s *Service) Parse(ctx context.Context, docID uuid.UUID) {
	_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusProcessing)

	doc, err := s.docRepo.Get(ctx, docID)
	if err != nil {
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	rc, err := s.bucket.Download(ctx, doc.StorageKey)
	if err != nil {
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}
	defer rc.Close()
	imageData, err := io.ReadAll(rc)
	if err != nil {
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	p, model, err := s.providerReg.ForRole("vision")
	if err != nil {
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	req := &provider.Request{
		Model:  model,
		System: parseDocumentSystemPrompt,
		Messages: []provider.Message{
			{Role: "user", Content: "Extract all lab values, medications, diagnoses and dates from this medical document. Return as JSON."},
		},
		Images:    []provider.Image{{Data: imageData, MimeType: doc.MimeType}},
		MaxTokens: 2000,
	}

	ch, err := p.Stream(ctx, req)
	if err != nil {
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	var sb strings.Builder
	for ev := range ch {
		if ev.Type == provider.EventTypeText {
			sb.WriteString(ev.Text)
		}
	}

	_ = sb.String() // TODO: parse JSON and insert lab_results rows
	_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusDone)
}

const parseDocumentSystemPrompt = `You are a medical document parser. Extract structured data from medical documents.
Return a JSON object with these fields:
- lab_results: array of {loinc_code, name, value, unit, reference_range, flag, date}
- medications: array of {name, dose, frequency, route}
- diagnoses: array of {icd10_code, name}
- document_date: ISO date string
Be precise. If you cannot determine a value, omit it.`
