package routes

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) handleDocumentUpload(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}

	visitID := c.FormValue("visit_id")
	sourceType := c.FormValue("source_type")
	if sourceType == "" {
		sourceType = "lab_result"
	}

	doc, err := s.docSvc.Upload(c.Context(), userID, visitID, file, sourceType)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Trigger async parse
	go s.docSvc.Parse(context.Background(), doc.ID)

	// Proactive: generate insight card for document upload
	if s.insightGenerator != nil {
		go s.insightGenerator.ProcessDataChange(context.Background(), userID, "document_uploaded", doc.ID.String())
	}

	return c.Status(202).JSON(doc)
}

func (s *Server) handleListDocuments(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}
	_ = userID
	return c.JSON(fiber.Map{"documents": []any{}})
}
