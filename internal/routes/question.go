package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) handleListQuestions(c *fiber.Ctx, userID uuid.UUID) error {
	unlinked := c.Query("unlinked") == "true"

	var questions []struct {
		ID        string  `json:"id"         db:"id"`
		Text      string  `json:"text"       db:"text"`
		Rationale string  `json:"rationale"  db:"rationale"`
		Urgency   string  `json:"urgency"    db:"urgency"`
		Source    string  `json:"source"     db:"source"`
		Asked     bool    `json:"asked"      db:"asked"`
		VisitID   *string `json:"visit_id"   db:"visit_id"`
		CreatedAt string  `json:"created_at" db:"created_at"`
	}

	var err error
	if unlinked {
		err = s.db.SelectContext(c.Context(), &questions, `
			SELECT id::text, text, rationale, urgency, source, asked, visit_id::text, created_at::text
			FROM question_backlog
			WHERE user_id = $1 AND visit_id IS NULL AND asked = FALSE
			ORDER BY created_at DESC
		`, userID)
	} else {
		err = s.db.SelectContext(c.Context(), &questions, `
			SELECT id::text, text, rationale, urgency, source, asked, visit_id::text, created_at::text
			FROM question_backlog
			WHERE user_id = $1 AND asked = FALSE
			ORDER BY created_at DESC
		`, userID)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if questions == nil {
		questions = make([]struct {
			ID        string  `json:"id"         db:"id"`
			Text      string  `json:"text"       db:"text"`
			Rationale string  `json:"rationale"  db:"rationale"`
			Urgency   string  `json:"urgency"    db:"urgency"`
			Source    string  `json:"source"     db:"source"`
			Asked     bool    `json:"asked"      db:"asked"`
			VisitID   *string `json:"visit_id"   db:"visit_id"`
			CreatedAt string  `json:"created_at" db:"created_at"`
		}, 0)
	}
	return c.JSON(questions)
}

func (s *Server) handleCreateQuestion(c *fiber.Ctx, userID uuid.UUID) error {
	var body struct {
		Text      string `json:"text"`
		Rationale string `json:"rationale"`
		Urgency   string `json:"urgency"`
	}
	if err := c.BodyParser(&body); err != nil || body.Text == "" {
		return c.Status(400).JSON(fiber.Map{"error": "text required"})
	}
	if body.Urgency == "" {
		body.Urgency = "routine"
	}

	var id string
	err := s.db.GetContext(c.Context(), &id, `
		INSERT INTO question_backlog (user_id, text, rationale, urgency, source)
		VALUES ($1, $2, $3, $4, 'manual')
		RETURNING id::text
	`, userID, body.Text, body.Rationale, body.Urgency)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"id": id, "text": body.Text, "rationale": body.Rationale, "urgency": body.Urgency})
}

func (s *Server) handleLinkQuestion(c *fiber.Ctx, userID uuid.UUID) error {
	qID := c.Params("id")
	var body struct {
		VisitID string `json:"visit_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.VisitID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "visit_id required"})
	}

	visitUUID, err := uuid.Parse(body.VisitID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid visit_id"})
	}

	// Verify visit belongs to user
	var visitOwner string
	err = s.db.GetContext(c.Context(), &visitOwner, `SELECT user_id::text FROM visits WHERE id = $1`, visitUUID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "visit not found"})
	}
	if visitOwner != userID.String() {
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}

	res, err := s.db.ExecContext(c.Context(), `
		UPDATE question_backlog SET visit_id = $1
		WHERE id = $2 AND user_id = $3 AND asked = FALSE
	`, visitUUID, qID, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "question not found"})
	}

	// Sync to visit plan_json
	syncBacklogQuestionToVisitPlan(c.Context(), s.db, visitUUID, qID)

	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) handleBulkLinkQuestions(c *fiber.Ctx, userID uuid.UUID) error {
	var body struct {
		QuestionIDs []string `json:"question_ids"`
		VisitID     string   `json:"visit_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.VisitID == "" || len(body.QuestionIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "visit_id and question_ids required"})
	}

	visitUUID, err := uuid.Parse(body.VisitID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid visit_id"})
	}

	// Verify visit belongs to user
	var visitOwner string
	err = s.db.GetContext(c.Context(), &visitOwner, `SELECT user_id::text FROM visits WHERE id = $1`, visitUUID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "visit not found"})
	}
	if visitOwner != userID.String() {
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}

	linked := 0
	for _, qid := range body.QuestionIDs {
		res, err := s.db.ExecContext(c.Context(), `
			UPDATE question_backlog SET visit_id = $1
			WHERE id = $2 AND user_id = $3 AND asked = FALSE
		`, visitUUID, qid, userID)
		if err == nil {
			n, _ := res.RowsAffected()
			linked += int(n)
		}
	}

	// Sync all to visit plan_json
	s.linkBacklogQuestions(c.Context(), userID, visitUUID)

	return c.JSON(fiber.Map{"linked": linked})
}
