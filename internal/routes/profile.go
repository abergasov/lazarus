package routes

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
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

func (s *Server) handleUpdateConditions(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewPatientModelRepo(s.db)
	model, err := repo.Load(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	var incoming []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(c.Body(), &incoming); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	model.ActiveConditions = make([]entities.Condition, 0, len(incoming))
	for _, c := range incoming {
		status := c.Status
		if status == "" {
			status = "active"
		}
		model.ActiveConditions = append(model.ActiveConditions, entities.Condition{
			ICD10Code:   "user-entered",
			Name:        c.Name,
			Status:      status,
			DiagnosedAt: time.Time{},
		})
	}
	if err := repo.Save(c.Context(), model); err != nil {
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
