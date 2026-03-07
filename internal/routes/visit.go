package routes

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

func (s *Server) handleCreateVisit(c *fiber.Ctx, userID uuid.UUID) error {
	var v entities.Visit
	if err := c.BodyParser(&v); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	v.UserID = userID

	repo := repository.NewVisitRepo(s.db)
	if err := repo.Create(c.Context(), &v); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(v)
}

func (s *Server) handleListVisits(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewVisitRepo(s.db)
	visits, err := repo.ListByUser(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if visits == nil {
		visits = []entities.Visit{}
	}
	return c.JSON(visits)
}

func (s *Server) handleGetVisit(c *fiber.Ctx, userID uuid.UUID) error {
	id := c.Params("id")
	repo := repository.NewVisitRepo(s.db)
	v, err := repo.Get(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "visit not found"})
	}
	if v.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.JSON(v)
}

func (s *Server) handleUpdateVisitPhase(c *fiber.Ctx, userID uuid.UUID) error {
	id := c.Params("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	repo := repository.NewVisitRepo(s.db)
	v, err := repo.Get(c.Context(), id)
	if err != nil || v.UserID != userID {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	if err := repo.UpdatePhase(c.Context(), id, body.Status); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Proactive: generate insight card for phase transition
	if s.insightGenerator != nil {
		go s.insightGenerator.ProcessDataChange(context.Background(), userID, "visit_phase_changed", id)
	}

	return c.JSON(fiber.Map{"status": body.Status})
}

// unused import guard
var _ = json.Marshal
