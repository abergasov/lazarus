package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

func (s *Server) handleListMedications(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewMedicationRepo(s.db)

	include := c.Query("include")
	var meds []entities.Medication
	var err error
	if include == "all" {
		meds, err = repo.ListAll(c.Context(), userID)
	} else {
		meds, err = repo.ListActive(c.Context(), userID)
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if meds == nil {
		meds = []entities.Medication{}
	}
	return c.JSON(meds)
}

func (s *Server) handleAddMedication(c *fiber.Ctx, userID uuid.UUID) error {
	var med entities.Medication
	if err := c.BodyParser(&med); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	med.UserID = userID

	// Default started_at to today if not provided
	if med.StartedAt == nil {
		now := time.Now()
		med.StartedAt = &now
	}

	repo := repository.NewMedicationRepo(s.db)
	if err := repo.Create(c.Context(), &med); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(med)
}

func (s *Server) handleDeleteMedication(c *fiber.Ctx, userID uuid.UUID) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	repo := repository.NewMedicationRepo(s.db)
	if err := repo.Deactivate(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(204).Send(nil)
}

func (s *Server) handleReactivateMedication(c *fiber.Ctx, userID uuid.UUID) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	repo := repository.NewMedicationRepo(s.db)
	if err := repo.Reactivate(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "reactivated"})
}
