# Document Detail Page — Design

## Goal

Let users view, verify, edit, and chat about their uploaded medical documents. The document is the source of truth — users must be able to see the original, cross-reference extracted data, fix OCR errors, and have AI-powered conversations about what's in each document.

## Architecture

New route `/app/documents/:id` with three tabs: **Original** | **Extracted** | **Chat**

### Data Flow

```
Upload → S3 (original file)
Parse:  PDF → pdftoppm → PNGs saved to S3 (page images) → LLM vision → structured data → DB
View:   Tab 1 reads PNGs from S3 via proxy endpoint
        Tab 2 reads extracted labs/meds from DB (linked via document_id)
        Tab 3 conversation with document context + full health history
Edit:   Tab 2 in edit mode → PUT/DELETE individual lab/med records
```

### Backend Changes

**Migration (000006):**
- Add `page_count INTEGER DEFAULT 0` to documents table
- Add `pages_key VARCHAR(500)` to documents table (S3 prefix for page images)

**Document service — Parse changes:**
- After pdftoppm, upload each page PNG to S3 at `documents/{user_id}/{doc_id}/pages/page-{N}.png`
- Store `pages_key` prefix and `page_count` on the document record
- Keep existing behavior otherwise

**New API endpoints:**
- `GET /api/v1/documents/:id` — Full document metadata (page_count, pages_key, etc.)
- `GET /api/v1/documents/:id/file` — Proxy original file from S3 (Content-Disposition: attachment)
- `GET /api/v1/documents/:id/pages/:num` — Proxy individual page PNG from S3
- `GET /api/v1/documents/:id/extracted` — Labs + meds linked to this document_id
- `PUT /api/v1/documents/:id/labs/:labId` — Update a lab result (name, value, unit, flag, date)
- `DELETE /api/v1/documents/:id/labs/:labId` — Delete a bad extraction

**Conversation context for documents:**
- `buildRichContext` for `context_type=document` loads:
  - Document metadata (filename, date, page count)
  - All labs extracted from this document
  - All meds extracted from this document
  - Full health summary (same as insight context)

### Frontend Changes

**New route: `ui/src/routes/app/documents/[id]/+page.svelte`**

**Tab 1: Original**
- Page images loaded from `/api/v1/documents/:id/pages/:num`
- Page navigation: prev/next buttons + "Page N of M" counter
- "Download original" button → `/api/v1/documents/:id/file`
- Skeleton placeholder while loading
- Image fits container width, scrollable vertically for tall pages

**Tab 2: Extracted**
- Two sections: Lab Results and Medications
- Lab row: name | value unit | flag badge | date
- Med row: name | dose | frequency
- "Edit" toggle button in header
- Edit mode:
  - Fields become inputs
  - Delete (X) button per row
  - Page thumbnail strip at top for cross-reference
  - "Save" and "Cancel" buttons replace "Edit"
- Empty state: "Nothing extracted" + "Re-process" button

**Tab 3: Chat**
- Embedded conversation (not overlay sheet)
- Same streaming SSE experience as ConversationSheet
- Context: this document's extracted data + full health history
- Persisted via `context_type=document, context_id=doc.id`

**Records page update:**
- Document row becomes clickable → navigates to `/app/documents/:id`

### API Types (Frontend)

```typescript
// Extended Document type
export type DocumentDetail = {
  id: string;
  file_name: string | null;
  mime_type: string | null;
  source_type: string;
  parse_status: string;
  page_count: number;
  created_at: string;
}

export type ExtractedData = {
  labs: Lab[];
  medications: Medication[];
}
```

### Design Principles

- Premium feel: smooth transitions, proper loading states, attention to spacing
- Trust through transparency: user sees original document alongside extracted data
- Edit mode is explicit: no accidental changes, clear save/cancel boundary
- Agent has full context: this document + entire health history
- One human, one story: agent understands all documents are pieces of one person's health journey
