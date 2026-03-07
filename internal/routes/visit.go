package routes

import (
	"context"
	"encoding/json"
	"time"

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

	// Auto-link unlinked backlog questions to this new visit
	s.linkBacklogQuestions(c.Context(), userID, v.ID)

	return c.Status(201).JSON(v)
}

// linkBacklogQuestions attaches all unlinked (visit_id IS NULL) questions to the new visit.
// It also syncs them into the visit's plan_json so the prep agent can see them.
func (s *Server) linkBacklogQuestions(ctx context.Context, userID, visitID uuid.UUID) {
	// Link the rows
	_, _ = s.db.ExecContext(ctx, `
		UPDATE question_backlog SET visit_id = $1
		WHERE user_id = $2 AND visit_id IS NULL AND asked = FALSE
	`, visitID, userID)

	// Sync to plan_json
	var questions []struct {
		Text      string `db:"text"`
		Rationale string `db:"rationale"`
	}
	_ = s.db.SelectContext(ctx, &questions, `
		SELECT text, rationale FROM question_backlog
		WHERE visit_id = $1 AND asked = FALSE
		ORDER BY created_at ASC
	`, visitID)

	if len(questions) == 0 {
		return
	}

	// Load or create plan
	var planRaw []byte
	_ = s.db.GetContext(ctx, &planRaw, `SELECT COALESCE(plan_json, '{}'::jsonb) FROM visits WHERE id = $1`, visitID)

	var plan entities.VisitPlan
	_ = json.Unmarshal(planRaw, &plan)

	maxRank := 0
	for _, q := range plan.Questions {
		if q.OrderRank > maxRank {
			maxRank = q.OrderRank
		}
	}

	for _, q := range questions {
		maxRank++
		plan.Questions = append(plan.Questions, entities.VisitQuestion{
			Text:      q.Text,
			Rationale: q.Rationale,
			OrderRank: maxRank,
			Asked:     false,
		})
	}

	if plan.GeneratedAt.IsZero() {
		plan.GeneratedAt = time.Now()
	}

	planJSON, _ := json.Marshal(plan)
	_, _ = s.db.ExecContext(ctx,
		`UPDATE visits SET plan_json = $1, updated_at = NOW() WHERE id = $2`,
		planJSON, visitID)
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
	v, err := repo.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "visit not found"})
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
	if _, err := repo.Get(c.Context(), id, userID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	if err := repo.UpdatePhase(c.Context(), id, userID, body.Status); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Proactive: generate insight card for phase transition
	if s.insightGenerator != nil {
		go s.insightGenerator.ProcessDataChange(context.Background(), userID, "visit_phase_changed", id)
	}

	return c.JSON(fiber.Map{"status": body.Status})
}

func (s *Server) handleUpdateVisitPlan(c *fiber.Ctx, userID uuid.UUID) error {
	id := c.Params("id")
	repo := repository.NewVisitRepo(s.db)
	_, err := repo.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	body := c.Body()
	if len(body) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "body required"})
	}

	if err := repo.UpdatePlan(c.Context(), id, userID, body); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) handleUpdateVisitOutcome(c *fiber.Ctx, userID uuid.UUID) error {
	id := c.Params("id")
	repo := repository.NewVisitRepo(s.db)
	_, err := repo.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	body := c.Body()
	if len(body) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "body required"})
	}

	if err := repo.UpdateOutcome(c.Context(), id, userID, body); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) handleAddVisitNote(c *fiber.Ctx, userID uuid.UUID) error {
	id := c.Params("id")
	repo := repository.NewVisitRepo(s.db)
	_, err := repo.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	var body struct {
		Text string `json:"text"`
	}
	if err := c.BodyParser(&body); err != nil || body.Text == "" {
		return c.Status(400).JSON(fiber.Map{"error": "text required"})
	}

	note := entities.VisitNote{Text: body.Text, Timestamp: time.Now()}
	if err := repo.AppendNote(c.Context(), id, userID, note); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(note)
}

func (s *Server) handleDeleteVisit(c *fiber.Ctx, userID uuid.UUID) error {
	id := c.Params("id")
	repo := repository.NewVisitRepo(s.db)
	_, err := repo.Get(c.Context(), id, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if err := repo.Delete(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

// unused import guard
var _ = json.Marshal
