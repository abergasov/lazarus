package document

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		MimeType:   &contentType,
		FileName:   &file.Filename,
		SizeBytes:  &file.Size,
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

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Document, error) {
	return s.docRepo.Get(ctx, id, userID)
}

func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]entities.Document, error) {
	return s.docRepo.ListByUser(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.docRepo.Delete(ctx, id, userID)
}

func (s *Service) DownloadFile(ctx context.Context, id uuid.UUID, userID uuid.UUID) (io.ReadCloser, string, string, error) {
	doc, err := s.docRepo.Get(ctx, id, userID)
	if err != nil {
		return nil, "", "", err
	}
	reader, err := s.bucket.Download(ctx, doc.StorageKey)
	if err != nil {
		return nil, "", "", err
	}
	mime := "application/octet-stream"
	if doc.MimeType != nil {
		mime = *doc.MimeType
	}
	name := "document"
	if doc.FileName != nil {
		name = *doc.FileName
	}
	return reader, mime, name, nil
}

// ReParsePending re-parses all documents stuck in pending status (e.g. after a failed parse).
func (s *Service) ReParsePending(ctx context.Context) {
	docs, err := s.docRepo.ListPending(ctx)
	if err != nil || len(docs) == 0 {
		return
	}
	slog.Info("reparse: found pending documents", "count", len(docs))
	for _, doc := range docs {
		go s.Parse(context.Background(), doc.ID)
	}
}

// parsedData is the expected JSON structure from the LLM OCR
type parsedData struct {
	LabResults  []parsedLab  `json:"lab_results"`
	Medications []parsedMed  `json:"medications"`
	Diagnoses   []parsedDiag `json:"diagnoses"`
	Date        string       `json:"date"`
	Category    string       `json:"category"`
	Specialty   string       `json:"specialty"`
	Summary     string       `json:"summary"`
}

type parsedLab struct {
	Name  string      `json:"name"`
	Value json.Number `json:"value"`
	Unit  string      `json:"unit"`
	Range string      `json:"range"`
	Flag  string      `json:"flag"`
	Date  string      `json:"date"`
}

type parsedMed struct {
	Name      string `json:"name"`
	Dose      string `json:"dose"`
	Frequency string `json:"frequency"`
}

type parsedDiag struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (s *Service) Parse(ctx context.Context, docID uuid.UUID) {
	_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusProcessing)

	doc, err := s.docRepo.GetInternal(ctx, docID)
	if err != nil {
		slog.Error("parse: get doc", "error", err, "doc_id", docID)
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	rc, err := s.bucket.Download(ctx, doc.StorageKey)
	if err != nil {
		slog.Error("parse: download", "error", err, "doc_id", docID)
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}
	defer rc.Close()
	fileData, err := io.ReadAll(rc)
	if err != nil {
		slog.Error("parse: read", "error", err, "doc_id", docID)
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	// Convert PDFs to images; pass images through directly
	images, mimeType, err := prepareImages(fileData, ptrStr(doc.MimeType))
	if err != nil {
		slog.Error("parse: prepare images", "error", err, "doc_id", docID)
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	p, model, err := s.providerReg.ForRole("vision")
	if err != nil {
		slog.Error("parse: provider", "error", err)
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	var provImages []provider.Image
	for _, img := range images {
		provImages = append(provImages, provider.Image{Data: img, MimeType: mimeType})
	}

	req := &provider.Request{
		Model:  model,
		System: parseDocumentSystemPrompt,
		Messages: []provider.Message{
			{Role: "user", Content: "Extract structured data from this document. JSON only, no commentary."},
		},
		Images:      provImages,
		MaxTokens:   4096,
		Temperature: 0,
	}

	ch, err := p.Stream(ctx, req)
	if err != nil {
		slog.Error("parse: stream", "error", err, "doc_id", docID)
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	var sb strings.Builder
	for ev := range ch {
		if ev.Type == provider.EventTypeText {
			sb.WriteString(ev.Text)
		}
		if ev.Type == provider.EventTypeError && ev.Error != nil {
			slog.Error("parse: LLM error", "error", ev.Error, "doc_id", docID)
			_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
			return
		}
	}

	raw := sb.String()
	slog.Info("parse: LLM response", "doc_id", docID, "length", len(raw))

	// Strip markdown fences if present
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw[3:], "\n"); idx >= 0 {
			raw = raw[3+idx+1:]
		}
		if strings.HasSuffix(raw, "```") {
			raw = raw[:len(raw)-3]
		}
		raw = strings.TrimSpace(raw)
	}

	var data parsedData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		slog.Error("parse: unmarshal LLM response", "error", err, "doc_id", docID, "raw", raw[:min(len(raw), 200)])
		_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusFailed)
		return
	}

	// Determine document date
	docDate := time.Now()
	if data.Date != "" {
		if t, err := time.Parse("2006-01-02", data.Date); err == nil {
			docDate = t
		}
	}

	labRepo := repository.NewLabRepo(s.db)
	medRepo := repository.NewMedicationRepo(s.db)

	labCount := 0
	for _, l := range data.LabResults {
		val, err := l.Value.Float64()
		if err != nil {
			continue
		}
		collectedAt := docDate
		if l.Date != "" {
			if t, err := time.Parse("2006-01-02", l.Date); err == nil {
				collectedAt = t
			}
		}
		flag := strings.ToLower(l.Flag)
		if flag == "" {
			flag = "normal"
		}
		lab := &entities.LabResult{
			UserID:      doc.UserID,
			DocumentID:  &doc.ID,
			Value:       val,
			Unit:        &l.Unit,
			Flag:        flag,
			LabName:     &l.Name,
			CollectedAt: collectedAt,
		}
		if err := labRepo.Insert(ctx, lab); err != nil {
			slog.Error("parse: insert lab", "error", err, "name", l.Name)
		} else {
			labCount++
		}
	}

	medCount := 0
	for _, m := range data.Medications {
		if m.Name == "" {
			continue
		}
		med := &entities.Medication{
			UserID:    doc.UserID,
			Name:      m.Name,
			Dose:      m.Dose,
			Frequency: m.Frequency,
		}
		if err := medRepo.Create(ctx, med); err != nil {
			slog.Error("parse: insert med", "error", err, "name", m.Name)
		} else {
			medCount++
		}
	}

	// Update document metadata from parsed content
	category := classifyDocument(data)
	s.docRepo.UpdateMeta(ctx, docID, category, data.Specialty, data.Summary, docDate)

	slog.Info("parse: complete", "doc_id", docID, "labs", labCount, "meds", medCount, "category", category)
	_ = s.docRepo.UpdateParseStatus(ctx, docID, entities.ParseStatusDone)
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// prepareImages converts document bytes to images suitable for the vision API.
// PDFs are rasterized to PNG pages via pdftoppm; images are returned as-is.
func prepareImages(data []byte, mimeType string) (pages [][]byte, outMime string, err error) {
	isPDF := mimeType == "application/pdf" ||
		(len(data) >= 5 && string(data[:5]) == "%PDF-")

	if !isPDF {
		// Already an image — return directly
		if mimeType == "" {
			mimeType = "image/png"
		}
		return [][]byte{data}, mimeType, nil
	}

	// Write PDF to temp file, convert with pdftoppm
	tmpDir, err := os.MkdirTemp("", "docparse-*")
	if err != nil {
		return nil, "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, data, 0600); err != nil {
		return nil, "", fmt.Errorf("write temp pdf: %w", err)
	}

	outPrefix := filepath.Join(tmpDir, "page")
	// -r 200: 200 DPI (good balance of quality vs size)
	// -l 10: max 10 pages to avoid huge documents
	// 60s timeout: prevents hung processes on malformed PDFs
	cmdCtx, cmdCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cmdCancel()
	cmd := exec.CommandContext(cmdCtx, "pdftoppm", "-png", "-r", "200", "-l", "10", pdfPath, outPrefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("pdftoppm: %w: %s", err, string(out))
	}

	// Read generated PNG files (pdftoppm names them page-01.png, page-02.png, etc.)
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, "", fmt.Errorf("read temp dir: %w", err)
	}

	var pngFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".png") {
			pngFiles = append(pngFiles, filepath.Join(tmpDir, e.Name()))
		}
	}
	sort.Strings(pngFiles) // ensure page order

	if len(pngFiles) == 0 {
		return nil, "", fmt.Errorf("pdftoppm produced no pages")
	}

	for _, f := range pngFiles {
		pageData, err := os.ReadFile(f)
		if err != nil {
			return nil, "", fmt.Errorf("read page %s: %w", f, err)
		}
		pages = append(pages, pageData)
	}

	return pages, "image/png", nil
}

// classifyDocument determines the document category from parsed content.
func classifyDocument(data parsedData) string {
	if data.Category != "" {
		// LLM provided a category — validate it
		switch data.Category {
		case "lab_result", "specialist_visit", "prescription", "imaging", "discharge", "referral", "vaccination", "insurance":
			return data.Category
		}
	}
	// Fallback heuristic
	if len(data.LabResults) > 0 {
		return "lab_result"
	}
	if len(data.Medications) > 0 && len(data.Diagnoses) == 0 {
		return "prescription"
	}
	if len(data.Diagnoses) > 0 {
		return "specialist_visit"
	}
	return "other"
}

const parseDocumentSystemPrompt = `Extract structured medical data from this document. Return ONLY a JSON object, no markdown fences, no explanation.

Schema:
{
  "lab_results": [{"name":"","value":0,"unit":"","range":"","flag":"normal|high|low","date":"YYYY-MM-DD"}],
  "medications": [{"name":"","dose":"","frequency":""}],
  "diagnoses": [{"code":"ICD-10","name":""}],
  "date": "YYYY-MM-DD",
  "category": "lab_result|specialist_visit|prescription|imaging|discharge|referral|vaccination|other",
  "specialty": "e.g. gastroenterology, urology, psychiatry, general_practice, endocrinology",
  "summary": "One-line summary of what this document is about"
}

Category guide:
- lab_result: blood work, urine analysis, any lab test results
- specialist_visit: doctor visit notes, examination results, consultation reports
- prescription: medication prescriptions
- imaging: X-ray, MRI, CT, ultrasound reports
- discharge: hospital discharge summaries
- referral: referral letters
- vaccination: vaccination records

Omit empty arrays. Always include date, category, specialty (if applicable), and summary.`
