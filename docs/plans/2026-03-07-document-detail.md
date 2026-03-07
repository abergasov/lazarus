# Document Detail Page — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let users view original documents, see/edit extracted data, and chat about each document.

**Architecture:** New route `/app/documents/:id` with three tabs (Original | Extracted | Chat). Backend adds migration for page storage, saves page PNGs during parse, adds proxy endpoints for file/page serving, extracted data endpoints, lab edit/delete, and document conversation context. Frontend creates a tabbed detail page with page image viewer, editable extracted data table, and embedded chat.

**Tech Stack:** Go 1.22 / Fiber v2, PostgreSQL 16, MinIO S3, SvelteKit 5 / Svelte 5 runes, `marked` for markdown

---

## Task 1: Database Migration (000006)

**Files:**
- Create: `migrations/000006_document_pages.up.sql`
- Create: `migrations/000006_document_pages.down.sql`

**Step 1: Write the up migration**

```sql
-- Add page storage metadata to documents
ALTER TABLE documents ADD COLUMN IF NOT EXISTS page_count INTEGER DEFAULT 0;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pages_key VARCHAR(500);
```

**Step 2: Write the down migration**

```sql
ALTER TABLE documents DROP COLUMN IF EXISTS pages_key;
ALTER TABLE documents DROP COLUMN IF EXISTS page_count;
```

**Step 3: Run migration**

```bash
cd /Users/igulida/.config/superpowers/worktrees/lazarus/medhelp-mvp
go run cmd/migrate/main.go up
```

Expected: Migration 6 applied successfully.

**Step 4: Update Document entity**

Modify: `internal/entities/document.go`

Add two fields to `Document` struct:

```go
PageCount int     `json:"page_count"  db:"page_count"`
PagesKey  *string `json:"pages_key"   db:"pages_key"`
```

**Step 5: Update DocumentRepo.Create insert statement**

Modify: `internal/repository/document.go:26-29`

Add `page_count` and `pages_key` to the INSERT columns and VALUES placeholders.

**Step 6: Commit**

```bash
git add migrations/ internal/entities/document.go internal/repository/document.go
git commit -m "feat: migration 000006 — add page_count and pages_key to documents"
```

---

## Task 2: Save Page PNGs to S3 During Parse

**Files:**
- Modify: `internal/service/document/service.go`
- Modify: `internal/repository/document.go`

**Step 1: Add `UpdatePages` to DocumentRepo**

Add to `internal/repository/document.go`:

```go
func (r *DocumentRepo) UpdatePages(ctx context.Context, id uuid.UUID, pagesKey string, pageCount int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET pages_key = $1, page_count = $2 WHERE id = $3`,
		pagesKey, pageCount, id)
	return err
}
```

**Step 2: Add `GetFile` and `GetPage` methods to document Service**

Add to `internal/service/document/service.go`:

```go
// Get returns a single document by ID, verifying ownership.
func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Document, error) {
	doc, err := s.docRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc.UserID != userID {
		return nil, fmt.Errorf("forbidden")
	}
	return doc, nil
}

// GetFile returns the original file from S3 as a ReadCloser.
func (s *Service) GetFile(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	return s.bucket.Download(ctx, storageKey)
}
```

**Step 3: Upload page PNGs to S3 in the Parse function**

In `internal/service/document/service.go`, inside the `Parse` method, after `prepareImages` succeeds and before the LLM call, add page upload logic:

```go
// Upload page images to S3 for later viewing
pagesPrefix := fmt.Sprintf("documents/%s/%s/pages", doc.UserID, doc.ID)
for i, pageData := range images {
	pageKey := fmt.Sprintf("%s/page-%d.png", pagesPrefix, i+1)
	if err := s.bucket.Upload(ctx, pageKey, bytes.NewReader(pageData), int64(len(pageData))); err != nil {
		slog.Error("parse: upload page", "error", err, "page", i+1, "doc_id", docID)
		// Non-fatal: continue with parsing even if page upload fails
	}
}
_ = s.docRepo.UpdatePages(ctx, docID, pagesPrefix, len(images))
```

This goes right after line 162 (`return` from prepareImages error), before `p, model, err := s.providerReg.ForRole("vision")`.

**Step 4: Verify Parse still compiles**

```bash
cd /Users/igulida/.config/superpowers/worktrees/lazarus/medhelp-mvp
go build ./...
```

**Step 5: Commit**

```bash
git add internal/service/document/service.go internal/repository/document.go
git commit -m "feat: upload page PNGs to S3 during document parse"
```

---

## Task 3: Backend API Endpoints

**Files:**
- Modify: `internal/routes/document.go` — add 5 new handlers
- Modify: `internal/routes/base.go` — register new routes
- Modify: `internal/repository/lab.go` — add GetByDocument, Update, Delete

**Step 1: Add lab repo methods**

Add to `internal/repository/lab.go`:

```go
func (r *LabRepo) ListByDocument(ctx context.Context, docID uuid.UUID, userID uuid.UUID) ([]entities.LabResult, error) {
	var labs []entities.LabResult
	err := r.db.SelectContext(ctx, &labs, `
		SELECT * FROM lab_results WHERE document_id = $1 AND user_id = $2 ORDER BY lab_name, collected_at
	`, docID, userID)
	return labs, err
}

func (r *LabRepo) Update(ctx context.Context, lab *entities.LabResult, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE lab_results SET lab_name = $1, value = $2, unit = $3, flag = $4, collected_at = $5
		WHERE id = $6 AND user_id = $7
	`, lab.LabName, lab.Value, lab.Unit, lab.Flag, lab.CollectedAt, lab.ID, userID)
	return err
}

func (r *LabRepo) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM lab_results WHERE id = $1 AND user_id = $2
	`, id, userID)
	return err
}
```

**Step 2: Add medication repo method to list by document**

The medications table doesn't have `document_id`. We need to add it or use a different approach. Looking at the existing schema: medications don't track which document they came from. For the extracted data view, we'll query all meds for the user. This matches reality since meds are deduplicated across documents.

No repo change needed for meds — we'll use the existing `ListActive`.

**Step 3: Write new route handlers**

Add to `internal/routes/document.go`:

```go
func (s *Server) handleGetDocument(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	doc, err := s.docSvc.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(doc)
}

func (s *Server) handleGetDocumentFile(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	doc, err := s.docSvc.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	rc, err := s.docSvc.GetFile(c.Context(), doc.StorageKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch file"})
	}
	defer rc.Close()

	if doc.MimeType != nil {
		c.Set("Content-Type", *doc.MimeType)
	}
	if doc.FileName != nil {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", *doc.FileName))
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read file"})
	}
	return c.Send(data)
}

func (s *Server) handleGetDocumentPage(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	pageNum := c.Params("num")
	doc, err := s.docSvc.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if doc.PagesKey == nil {
		return c.Status(404).JSON(fiber.Map{"error": "no pages available"})
	}
	pageKey := fmt.Sprintf("%s/page-%s.png", *doc.PagesKey, pageNum)
	rc, err := s.docSvc.GetFile(c.Context(), pageKey)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "page not found"})
	}
	defer rc.Close()
	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=86400")
	data, err := io.ReadAll(rc)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read page"})
	}
	return c.Send(data)
}

func (s *Server) handleGetDocumentExtracted(c *fiber.Ctx, userID uuid.UUID) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	labRepo := repository.NewLabRepo(s.db)
	labs, err := labRepo.ListByDocument(c.Context(), id, userID)
	if err != nil {
		labs = []entities.LabResult{}
	}
	medRepo := repository.NewMedicationRepo(s.db)
	meds, err := medRepo.ListActive(c.Context(), userID)
	if err != nil {
		meds = []entities.Medication{}
	}
	return c.JSON(fiber.Map{
		"labs":        labs,
		"medications": meds,
	})
}

func (s *Server) handleUpdateLab(c *fiber.Ctx, userID uuid.UUID) error {
	labID, err := uuid.Parse(c.Params("labId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid lab id"})
	}
	var body struct {
		LabName     *string  `json:"lab_name"`
		Value       float64  `json:"value"`
		Unit        *string  `json:"unit"`
		Flag        string   `json:"flag"`
		CollectedAt string   `json:"collected_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	collectedAt, err := time.Parse(time.RFC3339, body.CollectedAt)
	if err != nil {
		collectedAt, err = time.Parse("2006-01-02", body.CollectedAt)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid collected_at"})
		}
	}
	lab := &entities.LabResult{
		ID:          labID,
		LabName:     body.LabName,
		Value:       body.Value,
		Unit:        body.Unit,
		Flag:        body.Flag,
		CollectedAt: collectedAt,
	}
	labRepo := repository.NewLabRepo(s.db)
	if err := labRepo.Update(c.Context(), lab, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "updated"})
}

func (s *Server) handleDeleteLab(c *fiber.Ctx, userID uuid.UUID) error {
	labID, err := uuid.Parse(c.Params("labId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid lab id"})
	}
	labRepo := repository.NewLabRepo(s.db)
	if err := labRepo.Delete(c.Context(), labID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
```

Add required imports to `document.go`:
```go
import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)
```

**Step 4: Register routes**

Add to `internal/routes/base.go` in `initRoutes()`, after the existing document routes (line 140):

```go
api.Get("/documents/:id", s.wrapAuthUUID(s.handleGetDocument))
api.Get("/documents/:id/file", s.wrapAuthUUID(s.handleGetDocumentFile))
api.Get("/documents/:id/pages/:num", s.wrapAuthUUID(s.handleGetDocumentPage))
api.Get("/documents/:id/extracted", s.wrapAuthUUID(s.handleGetDocumentExtracted))
api.Put("/documents/:id/labs/:labId", s.wrapAuthUUID(s.handleUpdateLab))
api.Delete("/documents/:id/labs/:labId", s.wrapAuthUUID(s.handleDeleteLab))
```

**IMPORTANT:** The `api.Get("/documents/:id", ...)` route must be registered AFTER `api.Get("/documents", ...)` so Fiber doesn't confuse the two. Put these new routes right after `api.Post("/documents/:id/reparse", ...)`.

**Step 5: Build and verify**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add internal/routes/ internal/repository/lab.go
git commit -m "feat: document detail API endpoints — get, file, pages, extracted, lab edit/delete"
```

---

## Task 4: Document Conversation Context

**Files:**
- Modify: `internal/routes/conversation.go`

**Step 1: Add `"document"` case to `buildRichContext`**

Add to `internal/routes/conversation.go`, inside the `switch contextType` block (before `default:`):

```go
case "document":
	id, err := uuid.Parse(contextID)
	if err != nil {
		return ""
	}
	// Get document metadata
	var doc struct {
		FileName    *string    `db:"file_name"`
		SourceType  string     `db:"source_type"`
		DocumentDate *time.Time `db:"document_date"`
		PageCount   int        `db:"page_count"`
	}
	err = s.db.GetContext(ctx, &doc,
		`SELECT file_name, source_type, document_date, page_count FROM documents WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return ""
	}
	var b strings.Builder
	fname := "Unknown document"
	if doc.FileName != nil {
		fname = *doc.FileName
	}
	b.WriteString(fmt.Sprintf("The user is asking about this document: \"%s\" (%s", fname, doc.SourceType))
	if doc.DocumentDate != nil {
		b.WriteString(fmt.Sprintf(", dated %s", doc.DocumentDate.Format("2006-01-02")))
	}
	b.WriteString(fmt.Sprintf(", %d pages)\n\n", doc.PageCount))

	// Labs extracted from this document
	var docLabs []struct {
		LabName     *string   `db:"lab_name"`
		Value       float64   `db:"value"`
		Unit        *string   `db:"unit"`
		Flag        string    `db:"flag"`
		CollectedAt time.Time `db:"collected_at"`
	}
	_ = s.db.SelectContext(ctx, &docLabs,
		`SELECT lab_name, value, unit, flag, collected_at FROM lab_results WHERE document_id = $1 AND user_id = $2 ORDER BY lab_name`,
		id, userID)
	if len(docLabs) > 0 {
		b.WriteString(fmt.Sprintf("Lab results from this document (%d):\n", len(docLabs)))
		for _, l := range docLabs {
			name := "Unknown"
			if l.LabName != nil {
				name = *l.LabName
			}
			unit := ""
			if l.Unit != nil {
				unit = *l.Unit
			}
			flag := ""
			if l.Flag != "normal" && l.Flag != "" {
				flag = " [" + strings.ToUpper(l.Flag) + "]"
			}
			b.WriteString(fmt.Sprintf("- %s: %.2f %s%s (%s)\n", name, l.Value, unit, flag, l.CollectedAt.Format("2006-01-02")))
		}
		b.WriteString("\n")
	}

	// Full health context so the agent understands the bigger picture
	b.WriteString(s.buildHealthSummary(ctx, userID))
	return b.String()
```

**Step 2: Build and verify**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/routes/conversation.go
git commit -m "feat: document conversation context — agent sees extracted data + full health history"
```

---

## Task 5: Frontend Types and API Client

**Files:**
- Modify: `ui/src/lib/types.ts`
- Modify: `ui/src/lib/api.ts`

**Step 1: Add DocumentDetail type**

Add to `ui/src/lib/types.ts`:

```typescript
export type DocumentDetail = {
  id: string;
  file_name: string | null;
  mime_type: string | null;
  source_type: string;
  parse_status: string;
  page_count: number;
  pages_key: string | null;
  created_at: string;
};

export type ExtractedData = {
  labs: Lab[];
  medications: Medication[];
};
```

**Step 2: Add document detail API methods**

Add to the `documents` object in `ui/src/lib/api.ts`:

```typescript
get: (id: string) => req<DocumentDetail>(`/documents/${id}`),
extracted: (id: string) => req<ExtractedData>(`/documents/${id}/extracted`),
pageUrl: (id: string, page: number) => `/api/v1/documents/${id}/pages/${page}`,
fileUrl: (id: string) => `/api/v1/documents/${id}/file`,
updateLab: (docId: string, labId: string, body: Partial<Lab>) =>
  req<any>(`/documents/${docId}/labs/${labId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }),
deleteLab: (docId: string, labId: string) =>
  req<void>(`/documents/${docId}/labs/${labId}`, { method: 'DELETE' }),
```

**Step 3: Commit**

```bash
cd ui && git add src/lib/types.ts src/lib/api.ts
git commit -m "feat: frontend types and API client for document detail"
```

---

## Task 6: Document Detail Page — Tab Structure

**Files:**
- Create: `ui/src/routes/app/documents/[id]/+page.ts`
- Create: `ui/src/routes/app/documents/[id]/+page.svelte`

**Step 1: Create the page load function**

Create `ui/src/routes/app/documents/[id]/+page.ts`:

```typescript
export function load({ params }: { params: { id: string } }) {
  return { id: params.id };
}
```

**Step 2: Create the page component with tab structure**

Create `ui/src/routes/app/documents/[id]/+page.svelte`:

```svelte
<script lang="ts">
  import { documents, conversations as convApi } from '$lib/api';
  import type { DocumentDetail, ExtractedData, Lab, Medication } from '$lib/types';
  import SegmentedControl from '$lib/components/SegmentedControl.svelte';

  let { data } = $props();

  let doc = $state<DocumentDetail | null>(null);
  let extracted = $state<ExtractedData | null>(null);
  let loading = $state(true);
  let tab = $state(0);

  async function load() {
    loading = true;
    try {
      const [docRes, extRes] = await Promise.all([
        documents.get(data.id),
        documents.extracted(data.id),
      ]);
      doc = docRes;
      extracted = extRes;
    } catch (e) {
      console.error('Failed to load document', e);
    }
    loading = false;
  }

  $effect(() => { load(); });
</script>

<div class="page">
  {#if loading}
    <div class="center"><div class="spinner"></div></div>
  {:else if doc}
    <header class="doc-header">
      <a href="/app/records" class="back-link">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
        Records
      </a>
      <h1>{doc.file_name || 'Document'}</h1>
      <span class="doc-meta">{doc.source_type} &middot; {new Date(doc.created_at).toLocaleDateString()}</span>
    </header>

    <div class="seg-wrap">
      <SegmentedControl segments={['Original', 'Extracted', 'Chat']} selected={tab} onchange={(i) => { tab = i; }} />
    </div>

    <div class="tab-content">
      {#if tab === 0}
        <!-- Original tab: rendered in Task 7 -->
        <div class="placeholder">Original document viewer</div>
      {:else if tab === 1}
        <!-- Extracted tab: rendered in Task 8 -->
        <div class="placeholder">Extracted data</div>
      {:else}
        <!-- Chat tab: rendered in Task 9 -->
        <div class="placeholder">Document chat</div>
      {/if}
    </div>
  {:else}
    <div class="empty"><p>Document not found.</p></div>
  {/if}
</div>

<style>
  .page { max-width: 800px; margin: 0 auto; padding: 24px 20px 40px; }
  .center { display: flex; justify-content: center; padding: 60px 0; }
  .spinner { width: 32px; height: 32px; border: 3px solid var(--separator); border-top-color: var(--blue); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }

  .doc-header { margin-bottom: 20px; }
  .back-link {
    display: inline-flex; align-items: center; gap: 4px;
    font-size: 14px; color: var(--blue); text-decoration: none;
    margin-bottom: 8px;
  }
  .back-link:hover { text-decoration: underline; }
  h1 { font-size: 24px; font-weight: 700; margin: 0 0 4px; }
  .doc-meta { font-size: 13px; color: var(--text3); }
  .seg-wrap { margin-bottom: 20px; }
  .tab-content { min-height: 300px; }
  .placeholder { text-align: center; padding: 60px 20px; color: var(--text3); }
  .empty { text-align: center; padding: 40px 20px; color: var(--text2); }
</style>
```

**Step 3: Verify page loads in browser**

```bash
cd ui && npm run dev
```

Navigate to `/app/documents/<any-doc-id>` and verify the tab shell renders.

**Step 4: Commit**

```bash
git add ui/src/routes/app/documents/
git commit -m "feat: document detail page shell with tab navigation"
```

---

## Task 7: Original Document Tab — Page Image Viewer

**Files:**
- Modify: `ui/src/routes/app/documents/[id]/+page.svelte`

**Step 1: Replace the "Original" tab placeholder**

Replace the `<!-- Original tab -->` placeholder with:

```svelte
{#if tab === 0}
  <div class="original-tab">
    {#if doc.page_count > 0}
      <div class="page-nav">
        <button class="nav-btn" disabled={currentPage <= 1} onclick={() => currentPage--}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        <span class="page-counter">Page {currentPage} of {doc.page_count}</span>
        <button class="nav-btn" disabled={currentPage >= doc.page_count} onclick={() => currentPage++}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
        </button>
      </div>
      <div class="page-viewer">
        {#key currentPage}
          {#if pageLoading}
            <div class="page-skeleton"></div>
          {/if}
          <img
            src={documents.pageUrl(data.id, currentPage)}
            alt="Page {currentPage}"
            class="page-image"
            class:hidden={pageLoading}
            onload={() => pageLoading = false}
            onerror={() => pageLoading = false}
          />
        {/key}
      </div>
    {:else}
      <div class="no-pages">
        <p>No page images available.</p>
        <p class="hint">This document may need re-processing.</p>
      </div>
    {/if}
    <a href={documents.fileUrl(data.id)} class="download-btn" download>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
      Download original
    </a>
  </div>
{/if}
```

**Step 2: Add state variables for page navigation**

Add to the script section:

```typescript
let currentPage = $state(1);
let pageLoading = $state(true);

// Reset page loading when page changes
$effect(() => {
  currentPage;
  pageLoading = true;
});
```

**Step 3: Add styles for the Original tab**

```css
.original-tab { display: flex; flex-direction: column; gap: 16px; }
.page-nav { display: flex; align-items: center; justify-content: center; gap: 16px; }
.nav-btn {
  all: unset; cursor: pointer; padding: 8px; border-radius: 8px;
  color: var(--text2); transition: background 0.15s, color 0.15s;
}
.nav-btn:hover:not(:disabled) { background: var(--bg2); color: var(--text); }
.nav-btn:disabled { opacity: 0.3; cursor: default; }
.page-counter { font-size: 14px; color: var(--text2); font-variant-numeric: tabular-nums; }
.page-viewer {
  background: var(--bg2); border-radius: var(--radius); overflow: hidden;
  min-height: 400px; display: flex; justify-content: center;
}
.page-image { width: 100%; height: auto; display: block; }
.page-image.hidden { display: none; }
.page-skeleton {
  width: 100%; min-height: 600px;
  background: linear-gradient(90deg, var(--bg2) 25%, var(--bg) 50%, var(--bg2) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }
.no-pages { text-align: center; padding: 40px 20px; color: var(--text3); }
.download-btn {
  display: inline-flex; align-items: center; gap: 8px; align-self: center;
  padding: 10px 20px; border-radius: 10px; font-size: 14px; font-weight: 500;
  color: var(--blue); background: rgba(13,148,136,0.1); text-decoration: none;
  transition: background 0.15s;
}
.download-btn:hover { background: rgba(13,148,136,0.18); }
```

**Step 4: Test in browser**

Upload a PDF document, let it parse, then navigate to `/app/documents/<id>`. Verify:
- Page images load
- Prev/next pagination works
- Skeleton shows during loading
- Download link works

**Step 5: Commit**

```bash
git add ui/src/routes/app/documents/
git commit -m "feat: original document tab — page image viewer with navigation"
```

---

## Task 8: Extracted Data Tab with Edit Mode

**Files:**
- Modify: `ui/src/routes/app/documents/[id]/+page.svelte`

**Step 1: Add extracted tab state variables**

Add to script section:

```typescript
let editing = $state(false);
let editLabs = $state<Lab[]>([]);
let saving = $state(false);
let deletedLabIds = $state<string[]>([]);
```

**Step 2: Replace the "Extracted" tab placeholder**

```svelte
{:else if tab === 1}
  <div class="extracted-tab">
    <div class="section-header">
      <h2>Lab Results</h2>
      {#if !editing}
        <button class="edit-toggle" onclick={() => { editing = true; editLabs = structuredClone(extracted?.labs ?? []); deletedLabIds = []; }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
          Edit
        </button>
      {:else}
        <div class="edit-actions">
          <button class="cancel-btn" onclick={() => { editing = false; }} disabled={saving}>Cancel</button>
          <button class="save-btn" onclick={saveEdits} disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      {/if}
    </div>

    {#if editing}
      {#if editLabs.length === 0}
        <div class="empty-section"><p>No lab results extracted.</p></div>
      {:else}
        <div class="edit-list">
          {#each editLabs as lab, i}
            <div class="edit-row">
              <input class="edit-field name" bind:value={lab.lab_name} placeholder="Test name" />
              <input class="edit-field value" type="number" step="any" bind:value={lab.value} placeholder="Value" />
              <input class="edit-field unit" bind:value={lab.unit} placeholder="Unit" />
              <select class="edit-field flag" bind:value={lab.flag}>
                <option value="normal">Normal</option>
                <option value="high">High</option>
                <option value="low">Low</option>
                <option value="critical_high">Critical High</option>
                <option value="critical_low">Critical Low</option>
              </select>
              <button class="delete-row-btn" onclick={() => {
                deletedLabIds.push(lab.id);
                editLabs = editLabs.filter((_, j) => j !== i);
              }} title="Delete">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      {#if !extracted?.labs?.length}
        <div class="empty-section"><p>No lab results extracted from this document.</p></div>
      {:else}
        <div class="data-list">
          {#each extracted.labs as lab}
            <div class="data-row">
              <span class="data-name">{lab.lab_name || 'Unknown'}</span>
              <span class="data-value" class:flagged={lab.flag !== 'normal' && lab.flag !== ''}>
                {lab.value}
                <span class="data-unit">{lab.unit || ''}</span>
              </span>
              {#if lab.flag && lab.flag !== 'normal'}
                <span class="flag-badge" class:high={lab.flag.includes('high')} class:low={lab.flag.includes('low')}>
                  {lab.flag.replace('_', ' ')}
                </span>
              {/if}
              <span class="data-date">{new Date(lab.collected_at).toLocaleDateString()}</span>
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <div class="section-header meds-header">
      <h2>Medications</h2>
    </div>
    {#if !extracted?.medications?.length}
      <div class="empty-section"><p>No medications found.</p></div>
    {:else}
      <div class="data-list">
        {#each extracted.medications as med}
          <div class="data-row">
            <span class="data-name">{med.name}</span>
            <span class="data-value">{med.dose}</span>
            <span class="data-date">{med.frequency}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>
```

**Step 3: Add the save function**

```typescript
async function saveEdits() {
  saving = true;
  try {
    // Delete removed labs
    for (const id of deletedLabIds) {
      await documents.deleteLab(data.id, id);
    }
    // Update remaining labs
    for (const lab of editLabs) {
      await documents.updateLab(data.id, lab.id, {
        lab_name: lab.lab_name,
        value: lab.value,
        unit: lab.unit,
        flag: lab.flag,
        collected_at: lab.collected_at,
      });
    }
    // Reload extracted data
    extracted = await documents.extracted(data.id);
    editing = false;
  } catch (e) {
    console.error('Save failed', e);
  }
  saving = false;
}
```

**Step 4: Add extracted tab styles**

```css
.extracted-tab { display: flex; flex-direction: column; gap: 4px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.section-header h2 { font-size: 18px; font-weight: 600; margin: 0; }
.meds-header { margin-top: 24px; }

.edit-toggle {
  all: unset; cursor: pointer; display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 500; color: var(--blue); padding: 6px 12px;
  border-radius: 8px; transition: background 0.15s;
}
.edit-toggle:hover { background: rgba(13,148,136,0.1); }

.edit-actions { display: flex; gap: 8px; }
.cancel-btn, .save-btn {
  all: unset; cursor: pointer; font-size: 13px; font-weight: 500;
  padding: 6px 14px; border-radius: 8px; transition: background 0.15s;
}
.cancel-btn { color: var(--text2); }
.cancel-btn:hover { background: var(--bg2); }
.save-btn { color: white; background: var(--blue); }
.save-btn:hover { opacity: 0.9; }
.save-btn:disabled, .cancel-btn:disabled { opacity: 0.5; cursor: default; }

.data-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
.data-row {
  display: flex; align-items: center; gap: 12px; padding: 12px 16px;
  border-bottom: 1px solid var(--separator);
}
.data-row:last-child { border-bottom: none; }
.data-name { flex: 1; font-size: 14px; font-weight: 500; }
.data-value { font-size: 15px; font-weight: 600; font-variant-numeric: tabular-nums; }
.data-value.flagged { color: var(--red); }
.data-unit { font-size: 12px; font-weight: 400; color: var(--text3); }
.data-date { font-size: 12px; color: var(--text3); min-width: 80px; text-align: right; }
.flag-badge {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 6px;
  text-transform: uppercase; letter-spacing: 0.3px;
}
.flag-badge.high { color: var(--red); background: rgba(255,59,48,0.1); }
.flag-badge.low { color: var(--orange); background: rgba(255,149,0,0.1); }

.edit-list { display: flex; flex-direction: column; gap: 8px; }
.edit-row {
  display: flex; align-items: center; gap: 8px; padding: 8px 12px;
  background: var(--bg2); border-radius: var(--radius);
}
.edit-field {
  padding: 8px 10px; border-radius: 8px; border: 1px solid var(--separator);
  font-size: 13px; background: var(--bg); outline: none;
}
.edit-field:focus { border-color: var(--blue); }
.edit-field.name { flex: 2; }
.edit-field.value { width: 80px; }
.edit-field.unit { width: 60px; }
.edit-field.flag { width: 100px; }
.delete-row-btn {
  all: unset; cursor: pointer; padding: 4px; border-radius: 6px;
  color: var(--text3); transition: color 0.15s;
}
.delete-row-btn:hover { color: var(--red); }

.empty-section { text-align: center; padding: 24px 20px; color: var(--text3); font-size: 14px; }
```

**Step 5: Test in browser**

1. Navigate to a document with extracted labs
2. Verify lab/med display
3. Click Edit → verify inputs appear
4. Change a value, delete a row, save → verify changes persist
5. Cancel → verify changes discarded

**Step 6: Commit**

```bash
git add ui/src/routes/app/documents/
git commit -m "feat: extracted data tab with edit mode — view, edit, delete labs"
```

---

## Task 9: Document Chat Tab (Embedded Conversation)

**Files:**
- Modify: `ui/src/routes/app/documents/[id]/+page.svelte`

**Step 1: Add chat state variables**

```typescript
import { marked } from 'marked';

let convId = $state('');
let convMessages = $state<{ role: string; content: string; timestamp: string }[]>([]);
let chatInput = $state('');
let chatStreaming = $state(false);
let chatStreamText = $state('');
let chatEl: HTMLDivElement;

marked.setOptions({ breaks: true, gfm: true });
function renderMarkdown(text: string): string {
  return marked.parse(text, { async: false }) as string;
}
```

**Step 2: Add chat initialization**

```typescript
async function initChat() {
  const conv = await convApi.create('document', data.id);
  convId = conv.id;
  convMessages = conv.messages || [];
}

// Load chat when switching to chat tab
$effect(() => {
  if (tab === 2 && !convId) initChat();
});
```

**Step 3: Add chat send function**

```typescript
async function sendChat() {
  if (!chatInput.trim() || chatStreaming || !convId) return;
  const msg = chatInput.trim();
  chatInput = '';

  convMessages.push({ role: 'user', content: msg, timestamp: new Date().toISOString() });
  convMessages = convMessages;
  chatStreaming = true;
  chatStreamText = '';

  try {
    const res = await convApi.sendMessage(convId, msg);
    if (!res.ok || !res.body) { chatStreaming = false; return; }
    const reader = res.body.getReader();
    const dec = new TextDecoder();
    let buf = '';
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buf += dec.decode(value, { stream: true });
      const parts = buf.split('\n\n');
      buf = parts.pop() ?? '';
      for (const part of parts) {
        let dataLine = '';
        for (const line of part.split('\n')) {
          if (line.startsWith('data:')) dataLine = line.slice(5).trim();
        }
        if (!dataLine || dataLine === '[DONE]') continue;
        try {
          const ev = JSON.parse(dataLine);
          if (ev.type === 'text_delta') chatStreamText += ev.payload?.text ?? '';
        } catch {}
      }
    }
    if (chatStreamText) {
      convMessages.push({ role: 'assistant', content: chatStreamText, timestamp: new Date().toISOString() });
      convMessages = convMessages;
    }
  } finally {
    chatStreaming = false;
    chatStreamText = '';
  }
}

function scrollChat() {
  if (chatEl) chatEl.scrollTop = chatEl.scrollHeight;
}

$effect(() => { if (convMessages.length || chatStreamText) setTimeout(scrollChat, 50); });
```

**Step 4: Replace the Chat tab placeholder**

```svelte
{:else}
  <div class="chat-tab">
    <div class="chat-messages" bind:this={chatEl}>
      {#each convMessages as msg}
        <div class="msg" class:user={msg.role === 'user'} class:assistant={msg.role === 'assistant'}>
          {#if msg.role === 'assistant'}
            <div class="msg-bubble prose">{@html renderMarkdown(msg.content)}</div>
          {:else}
            <div class="msg-bubble">{msg.content}</div>
          {/if}
        </div>
      {/each}
      {#if chatStreaming && chatStreamText}
        <div class="msg assistant">
          <div class="msg-bubble prose streaming">{@html renderMarkdown(chatStreamText)}</div>
        </div>
      {/if}
      {#if chatStreaming && !chatStreamText}
        <div class="msg assistant">
          <div class="msg-bubble streaming thinking">Thinking...</div>
        </div>
      {/if}
      {#if !convId}
        <div class="chat-loading"><div class="spinner small"></div></div>
      {:else if convMessages.length === 0 && !chatStreaming}
        <div class="chat-empty">
          <p>Ask anything about this document.</p>
          <p class="hint">The agent can see the extracted data and your full health history.</p>
        </div>
      {/if}
    </div>
    <form class="chat-input-row" onsubmit={(e) => { e.preventDefault(); sendChat(); }}>
      <input bind:value={chatInput} placeholder="Ask about this document..." disabled={chatStreaming || !convId} />
      <button type="submit" disabled={chatStreaming || !chatInput.trim() || !convId}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
      </button>
    </form>
  </div>
{/if}
```

**Step 5: Add chat styles**

```css
.chat-tab {
  display: flex; flex-direction: column;
  height: calc(100vh - 280px); min-height: 400px;
}
.chat-messages {
  flex: 1; overflow-y: auto; padding: 16px 0;
  display: flex; flex-direction: column; gap: 12px;
}
.msg { display: flex; }
.msg.user { justify-content: flex-end; }
.msg-bubble {
  max-width: 85%; padding: 10px 14px; border-radius: 16px;
  font-size: 14px; line-height: 1.5;
}
.msg.user .msg-bubble {
  background: var(--blue); color: white; border-bottom-right-radius: 4px;
}
.msg.assistant .msg-bubble {
  background: var(--bg2); color: var(--text); border-bottom-left-radius: 4px;
}
.msg-bubble.prose :global(p) { margin: 0 0 8px 0; }
.msg-bubble.prose :global(p:last-child) { margin-bottom: 0; }
.msg-bubble.prose :global(strong) { font-weight: 600; }
.msg-bubble.prose :global(ul), .msg-bubble.prose :global(ol) { margin: 4px 0 8px; padding-left: 18px; }
.msg-bubble.prose :global(li) { margin-bottom: 2px; }
.msg-bubble.thinking { color: var(--text3); animation: pulse 1.5s ease-in-out infinite; }
@keyframes pulse { 0%, 100% { opacity: 0.5; } 50% { opacity: 1; } }

.chat-input-row {
  display: flex; gap: 8px; padding: 12px 0;
  border-top: 1px solid var(--separator);
}
.chat-input-row input {
  flex: 1; padding: 10px 16px; border-radius: 20px;
  border: 1px solid var(--separator); font-size: 14px;
  outline: none; background: var(--bg2);
}
.chat-input-row input:focus { border-color: var(--blue); }
.chat-input-row button {
  all: unset; cursor: pointer; padding: 8px; color: var(--blue); transition: opacity 0.15s;
}
.chat-input-row button:disabled { opacity: 0.3; cursor: default; }

.chat-loading { display: flex; justify-content: center; padding: 40px 0; }
.spinner.small { width: 24px; height: 24px; }
.chat-empty { text-align: center; padding: 40px 20px; color: var(--text3); }
```

**Step 6: Test in browser**

1. Navigate to document detail → Chat tab
2. Send a message asking about the document
3. Verify streaming response with markdown
4. Verify conversation persists (refresh page, switch tabs)

**Step 7: Commit**

```bash
git add ui/src/routes/app/documents/
git commit -m "feat: document chat tab — embedded conversation with streaming + markdown"
```

---

## Task 10: Make Document Rows Clickable

**Files:**
- Modify: `ui/src/routes/app/records/+page.svelte`

**Step 1: Make document rows navigate to detail page**

In `ui/src/routes/app/records/+page.svelte`, change the doc row from a `<div>` to an `<a>` tag:

Replace lines 138-155 (the `{#each docList as doc}` block). Change:

```svelte
<div class="doc-row">
```

to:

```svelte
<a href="/app/documents/{doc.id}" class="doc-row">
```

And change the closing `</div>` to `</a>`.

**Step 2: Add navigation styling**

Add to the `<style>` block:

```css
a.doc-row { text-decoration: none; color: inherit; cursor: pointer; transition: background 0.15s; }
a.doc-row:hover { background: var(--bg); }
```

**Step 3: Prevent event bubbling on action buttons**

Add `onclick|stopPropagation` to the reparse and delete buttons inside doc rows (so clicking them doesn't navigate):

```svelte
<button class="doc-action" title="Re-process" onclick={(e) => { e.preventDefault(); e.stopPropagation(); reparseDoc(doc.id); }}>
```

```svelte
<button class="doc-action delete" title="Delete" onclick={(e) => { e.preventDefault(); e.stopPropagation(); deleteDoc(doc.id); }}>
```

**Step 4: Test**

1. Go to Records → Documents tab
2. Click a document row → should navigate to `/app/documents/:id`
3. Click reparse/delete → should NOT navigate

**Step 5: Commit**

```bash
git add ui/src/routes/app/records/+page.svelte
git commit -m "feat: make document rows clickable — navigate to detail page"
```

---

## Task 11: Build, Test, and Final Verification

**Step 1: Rebuild backend**

```bash
cd /Users/igulida/.config/superpowers/worktrees/lazarus/medhelp-mvp
go build ./...
```

**Step 2: Rebuild frontend (no cache)**

```bash
cd ui
rm -rf .svelte-kit node_modules/.vite
npm run build
```

**Step 3: Run the full app**

```bash
cd /Users/igulida/.config/superpowers/worktrees/lazarus/medhelp-mvp
# Start with docker compose or however the app runs
```

**Step 4: End-to-end test**

1. Upload a multi-page PDF document
2. Wait for it to process (status → "Processed")
3. Click the document row → detail page opens
4. **Original tab**: Page images display, prev/next works, download works
5. **Extracted tab**: Labs and meds show correctly
6. **Edit mode**: Toggle edit, change a value, delete a row, save, verify changes persist
7. **Chat tab**: Ask about the document, verify streaming response, verify agent has context

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: document detail page — complete with original viewer, extracted data edit, and chat"
```
