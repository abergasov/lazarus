package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

type HomeResponse struct {
	PrimaryCard         *entities.InsightCard  `json:"primary_card"`
	Visits              []entities.Visit       `json:"visits"`
	Insights            []entities.InsightCard `json:"insights"`
	OnboardingCompleted bool                   `json:"onboarding_completed"`
}

func (s *Server) handleHome(c *fiber.Ctx, userID uuid.UUID) error {
	resp := HomeResponse{
		Visits:   []entities.Visit{},
		Insights: []entities.InsightCard{},
	}

	// Check onboarding status
	pmRepo := repository.NewPatientModelRepo(s.db)
	model, err := pmRepo.Load(c.Context(), userID)
	if err == nil && model != nil {
		resp.OnboardingCompleted = model.OnboardingCompleted
	}

	// Load visits
	visitRepo := repository.NewVisitRepo(s.db)
	visits, err := visitRepo.ListByUser(c.Context(), userID)
	if err == nil && visits != nil {
		resp.Visits = visits
	}

	// Load active insights
	icRepo := repository.NewInsightCardRepo(s.db)
	insights, err := icRepo.ListActive(c.Context(), userID)
	if err == nil && insights != nil {
		resp.Insights = insights
		// Primary card = highest severity, most recent
		if len(insights) > 0 {
			resp.PrimaryCard = &insights[0]
			// Prefer urgent > warning > info
			for i := range insights {
				if insights[i].Severity == entities.SeverityUrgent {
					resp.PrimaryCard = &insights[i]
					break
				}
				if insights[i].Severity == entities.SeverityWarning && resp.PrimaryCard.Severity != entities.SeverityUrgent {
					resp.PrimaryCard = &insights[i]
				}
			}
		}
	}

	return c.JSON(resp)
}
