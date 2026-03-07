package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

type PendingQuestion struct {
	Text       string `json:"text"`
	Rationale  string `json:"rationale"`
	VisitID    string `json:"visit_id"`
	DoctorName string `json:"doctor_name"`
	VisitDate  string `json:"visit_date"`
}

type HomeResponse struct {
	PrimaryCard         *entities.InsightCard  `json:"primary_card"`
	Visits              []entities.Visit       `json:"visits"`
	Insights            []entities.InsightCard `json:"insights"`
	PendingQuestions    []PendingQuestion      `json:"pending_questions"`
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

	// Aggregate pending questions from backlog table (single source of truth)
	resp.PendingQuestions = []PendingQuestion{}
	var backlogQuestions []struct {
		Text       string  `db:"text"`
		Rationale  string  `db:"rationale"`
		VisitID    *string `db:"visit_id"`
		DoctorName *string `db:"doctor_name"`
		VisitDate  *string `db:"visit_date"`
	}
	_ = s.db.SelectContext(c.Context(), &backlogQuestions, `
		SELECT q.text, q.rationale,
		       q.visit_id::text,
		       v.doctor_name,
		       CASE WHEN v.visit_date IS NOT NULL THEN TO_CHAR(v.visit_date, 'YYYY-MM-DD') END AS visit_date
		FROM question_backlog q
		LEFT JOIN visits v ON v.id = q.visit_id
		WHERE q.user_id = $1 AND q.asked = FALSE AND q.visit_id IS NULL
		ORDER BY q.created_at DESC
	`, userID)
	for _, q := range backlogQuestions {
		pq := PendingQuestion{
			Text:      q.Text,
			Rationale: q.Rationale,
		}
		if q.VisitID != nil {
			pq.VisitID = *q.VisitID
		}
		if q.DoctorName != nil {
			pq.DoctorName = *q.DoctorName
		}
		if q.VisitDate != nil {
			pq.VisitDate = *q.VisitDate
		}
		resp.PendingQuestions = append(resp.PendingQuestions, pq)
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
