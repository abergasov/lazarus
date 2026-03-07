package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/repository"
)

func (s *Server) handleListInsights(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewInsightCardRepo(s.db)
	cards, err := repo.ListActive(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cards)
}

func (s *Server) handleDismissInsight(c *fiber.Ctx, userID uuid.UUID) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	repo := repository.NewInsightCardRepo(s.db)

	// Verify ownership via user-scoped query
	_, err = repo.GetByID(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	if err := repo.Dismiss(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
