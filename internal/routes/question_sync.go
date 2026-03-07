package routes

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

// syncBacklogQuestionToVisitPlan adds a single backlog question to a visit's plan_json.
func syncBacklogQuestionToVisitPlan(ctx context.Context, db *sqlx.DB, visitID uuid.UUID, questionID string) {
	var q struct {
		Text      string `db:"text"`
		Rationale string `db:"rationale"`
	}
	if err := db.GetContext(ctx, &q, `SELECT text, rationale FROM question_backlog WHERE id = $1`, questionID); err != nil {
		return
	}

	var planRaw []byte
	if err := db.GetContext(ctx, &planRaw, `SELECT COALESCE(plan_json, '{}'::jsonb) FROM visits WHERE id = $1`, visitID); err != nil {
		return
	}

	var plan entities.VisitPlan
	_ = json.Unmarshal(planRaw, &plan)

	maxRank := 0
	for _, existing := range plan.Questions {
		if existing.OrderRank > maxRank {
			maxRank = existing.OrderRank
		}
	}

	plan.Questions = append(plan.Questions, entities.VisitQuestion{
		Text:      q.Text,
		Rationale: q.Rationale,
		OrderRank: maxRank + 1,
		Asked:     false,
	})

	if plan.GeneratedAt.IsZero() {
		plan.GeneratedAt = time.Now()
	}

	planJSON, _ := json.Marshal(plan)
	_, _ = db.ExecContext(ctx,
		`UPDATE visits SET plan_json = $1, updated_at = NOW() WHERE id = $2`,
		planJSON, visitID)
}
