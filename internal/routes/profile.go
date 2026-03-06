package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/repository"
)

func (s *Server) handleGetProfile(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewPatientModelRepo(s.db)
	model, err := repo.Load(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(model)
}

func (s *Server) handleUpdateDemographics(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewPatientModelRepo(s.db)
	model, err := repo.Load(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if err := c.BodyParser(&model.Demographics); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := repo.Save(c.Context(), model); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(model)
}
