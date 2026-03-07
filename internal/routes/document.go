package routes

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
)

func (s *Server) handleDocumentUpload(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}

	visitID := c.FormValue("visit_id")
	sourceType := c.FormValue("source_type")
	if sourceType == "" {
		sourceType = "lab_result"
	}

	// Support multiple files: try "files" (multi) then fallback to "file" (single)
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "multipart form required"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "at least one file required"})
	}

	docs := make([]any, 0, len(files))
	for _, file := range files {
		doc, err := s.docSvc.Upload(c.Context(), userID, visitID, file, sourceType)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		go s.docSvc.Parse(context.Background(), doc.ID)

		if s.insightGenerator != nil {
			go s.insightGenerator.ProcessDataChange(context.Background(), userID, "document_uploaded", doc.ID.String())
		}

		docs = append(docs, doc)
	}

	// Return single doc for backward compat, array if multiple
	if len(docs) == 1 {
		return c.Status(202).JSON(docs[0])
	}
	return c.Status(202).JSON(fiber.Map{"documents": docs})
}

func (s *Server) handleDeleteDocument(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := s.docSvc.Delete(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (s *Server) handleReparseDocument(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	go s.docSvc.Parse(context.Background(), id)
	return c.JSON(fiber.Map{"status": "reparsing"})
}

func (s *Server) handleListDocuments(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}
	docs, err := s.docSvc.ListByUser(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if docs == nil {
		docs = []entities.Document{}
	}
	return c.JSON(docs)
}
