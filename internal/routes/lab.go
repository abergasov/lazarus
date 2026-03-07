package routes

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

func (s *Server) handleListLabs(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewLabRepo(s.db)
	labs, err := repo.ListByUser(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if labs == nil {
		labs = []entities.LabResult{}
	}
	return c.JSON(labs)
}

func (s *Server) handleListLabsByDocument(c *fiber.Ctx, userID uuid.UUID) error {
	docID, err := uuid.Parse(c.Params("docId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid document id"})
	}
	repo := repository.NewLabRepo(s.db)
	labs, err := repo.ListByDocument(c.Context(), userID, docID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if labs == nil {
		labs = []entities.LabResult{}
	}
	return c.JSON(labs)
}

type CreateLabRequest struct {
	LabName     string   `json:"lab_name"`
	Value       float64  `json:"value"`
	Unit        string   `json:"unit"`
	Flag        string   `json:"flag"`
	CollectedAt string   `json:"collected_at"`
	DocumentID  string   `json:"document_id"`
}

func (s *Server) handleCreateLab(c *fiber.Ctx, userID uuid.UUID) error {
	var req CreateLabRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.LabName == "" || req.Value == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "lab_name and value are required"})
	}
	collectedAt := time.Now()
	if req.CollectedAt != "" {
		if t, err := time.Parse(time.RFC3339, req.CollectedAt); err == nil {
			collectedAt = t
		} else if t, err := time.Parse("2006-01-02", req.CollectedAt); err == nil {
			collectedAt = t
		}
	}
	if req.Flag == "" {
		req.Flag = "normal"
	}
	lab := &entities.LabResult{
		UserID:      userID,
		Value:       req.Value,
		Unit:        &req.Unit,
		Flag:        req.Flag,
		LabName:     &req.LabName,
		CollectedAt: collectedAt,
	}
	if req.DocumentID != "" {
		docID, err := uuid.Parse(req.DocumentID)
		if err == nil {
			lab.DocumentID = &docID
		}
	}
	repo := repository.NewLabRepo(s.db)
	if err := repo.Insert(c.Context(), lab); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(lab)
}

type UpdateLabRequest struct {
	LabName     string  `json:"lab_name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Flag        string  `json:"flag"`
	CollectedAt string  `json:"collected_at"`
}

func (s *Server) handleUpdateLab(c *fiber.Ctx, userID uuid.UUID) error {
	labID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid lab id"})
	}
	var req UpdateLabRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	collectedAt, err := time.Parse(time.RFC3339, req.CollectedAt)
	if err != nil {
		collectedAt, err = time.Parse("2006-01-02", req.CollectedAt)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid collected_at"})
		}
	}
	repo := repository.NewLabRepo(s.db)
	lab := &entities.LabResult{
		ID:          labID,
		UserID:      userID,
		LabName:     &req.LabName,
		Value:       req.Value,
		Unit:        &req.Unit,
		Flag:        req.Flag,
		CollectedAt: collectedAt,
	}
	if err := repo.Update(c.Context(), lab); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) handleDeleteLab(c *fiber.Ctx, userID uuid.UUID) error {
	labID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid lab id"})
	}
	repo := repository.NewLabRepo(s.db)
	if err := repo.Delete(c.Context(), labID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (s *Server) handleLabTrend(c *fiber.Ctx, userID uuid.UUID) error {
	loincCode := c.Params("loinc")
	months := 24
	if m := c.Query("months"); m != "" {
		if v, err := strconv.Atoi(m); err == nil {
			months = v
		}
	}

	repo := repository.NewLabRepo(s.db)
	pts, err := repo.GetTrend(c.Context(), userID, loincCode, months)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if s.labSvc != nil {
		trend := s.labSvc.CalculateTrend(loincCode, loincCode, pts)
		return c.JSON(trend)
	}
	return c.JSON(pts)
}
