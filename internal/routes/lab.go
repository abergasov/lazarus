package routes

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/repository"
)

func (s *Server) handleListLabs(c *fiber.Ctx, userID uuid.UUID) error {
	repo := repository.NewLabRepo(s.db)
	labs, err := repo.ListByUser(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(labs)
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
