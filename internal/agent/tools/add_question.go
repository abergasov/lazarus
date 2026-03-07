package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"lazarus/internal/entities"

	"github.com/google/uuid"
)

func addDoctorQuestionTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":     "object",
		"required": []string{"question", "rationale"},
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The question to ask the doctor, written in the patient's voice",
			},
			"rationale": map[string]any{
				"type":        "string",
				"description": "Why this question is important for the patient",
			},
			"urgency": map[string]any{
				"type":        "string",
				"enum":        []string{"critical", "high", "routine"},
				"description": "How urgent this question is (default: routine)",
			},
		},
	})

	return &Tool{
		Name:        "add_doctor_question",
		Description: "Add a question to the patient's doctor visit backlog. Works even without an existing visit — the question will be saved and automatically linked when a visit is created. Use this when you identify something the patient should discuss with their doctor.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseGeneral},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var payload struct {
				Question  string `json:"question"`
				Rationale string `json:"rationale"`
				Urgency   string `json:"urgency"`
			}
			if err := json.Unmarshal(args, &payload); err != nil {
				return nil, fmt.Errorf("parse args: %w", err)
			}
			if payload.Urgency == "" {
				payload.Urgency = "routine"
			}

			if deps.DB == nil {
				return map[string]string{"status": "saved", "note": "no database"}, nil
			}

			userID, err := uuid.Parse(uc.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user ID: %w", err)
			}

			// Check for duplicates in the backlog
			questionLower := strings.ToLower(strings.TrimSpace(payload.Question))
			var existingCount int
			_ = deps.DB.GetContext(ctx, &existingCount, `
				SELECT COUNT(*) FROM question_backlog
				WHERE user_id = $1 AND asked = FALSE AND LOWER(TRIM(text)) = $2
			`, userID, questionLower)
			if existingCount > 0 {
				return map[string]string{
					"status": "duplicate",
					"note":   "This question is already in the backlog.",
				}, nil
			}

			// Fuzzy dedup: check word overlap against existing unasked questions
			var existingTexts []string
			_ = deps.DB.SelectContext(ctx, &existingTexts, `
				SELECT text FROM question_backlog
				WHERE user_id = $1 AND asked = FALSE
			`, userID)
			for _, existing := range existingTexts {
				if wordOverlap(questionLower, strings.ToLower(existing)) > 0.8 {
					return map[string]string{
						"status": "duplicate",
						"note":   "A very similar question already exists in the backlog.",
					}, nil
				}
			}

			// Find the visit to link to (if any)
			var visitID *uuid.UUID
			if uc.VisitID != "" {
				vid, err := uuid.Parse(uc.VisitID)
				if err == nil {
					visitID = &vid
				}
			}
			if visitID == nil {
				// Try to find an upcoming visit
				var nextVisitID string
				err := deps.DB.GetContext(ctx, &nextVisitID, `
					SELECT id FROM visits
					WHERE user_id = $1 AND status IN ('preparing', 'during')
					ORDER BY
						CASE WHEN visit_date >= NOW() THEN 0 ELSE 1 END,
						visit_date ASC
					LIMIT 1
				`, userID)
				if err == nil {
					vid, _ := uuid.Parse(nextVisitID)
					visitID = &vid
				}
				// If no visit found, visitID stays nil — question saved to backlog without a visit
			}

			// Save to backlog table
			_, err = deps.DB.ExecContext(ctx, `
				INSERT INTO question_backlog (user_id, visit_id, text, rationale, urgency, source)
				VALUES ($1, $2, $3, $4, $5, 'agent')
			`, userID, visitID, payload.Question, payload.Rationale, payload.Urgency)
			if err != nil {
				return nil, fmt.Errorf("save question: %w", err)
			}

			// Also sync to visit plan_json for backwards compatibility (if visit exists)
			if visitID != nil {
				syncQuestionToVisitPlan(ctx, deps, *visitID, payload.Question, payload.Rationale)
			}

			status := "added"
			note := "Question saved to your backlog"
			if visitID != nil {
				note += " and linked to your upcoming visit"
			} else {
				note += " — it will be linked when you create a visit"
			}

			return map[string]string{
				"status": status,
				"note":   note,
			}, nil
		},
		HumanLabel: func(_ json.RawMessage) string { return "Adding question to visit plan..." },
		ResultSummary: func(r any) string {
			if m, ok := r.(map[string]string); ok {
				if m["status"] == "duplicate" {
					return "Already in your backlog"
				}
				return "Question added"
			}
			return "Done"
		},
	}
}

// syncQuestionToVisitPlan keeps the visit plan_json in sync with the backlog
func syncQuestionToVisitPlan(ctx context.Context, deps *Deps, visitID uuid.UUID, text, rationale string) {
	var planRaw []byte
	err := deps.DB.GetContext(ctx, &planRaw, `SELECT COALESCE(plan_json, '{}'::jsonb) FROM visits WHERE id = $1`, visitID)
	if err != nil {
		return
	}

	var plan entities.VisitPlan
	_ = json.Unmarshal(planRaw, &plan)

	maxRank := 0
	for _, q := range plan.Questions {
		if q.OrderRank > maxRank {
			maxRank = q.OrderRank
		}
	}

	plan.Questions = append(plan.Questions, entities.VisitQuestion{
		Text:      text,
		Rationale: rationale,
		OrderRank: maxRank + 1,
		Asked:     false,
	})

	if plan.GeneratedAt.IsZero() {
		plan.GeneratedAt = time.Now()
	}

	planJSON, _ := json.Marshal(plan)
	_, _ = deps.DB.ExecContext(ctx,
		`UPDATE visits SET plan_json = $1, updated_at = NOW() WHERE id = $2`,
		planJSON, visitID)
}

// wordOverlap calculates the fraction of words in a that also appear in b
func wordOverlap(a, b string) float64 {
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	bSet := make(map[string]bool, len(wordsB))
	for _, w := range wordsB {
		bSet[w] = true
	}
	matches := 0
	for _, w := range wordsA {
		if bSet[w] {
			matches++
		}
	}
	return float64(matches) / float64(len(wordsA))
}
