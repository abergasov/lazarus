package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"lazarus/internal/entities"
)

func savePlanTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":     "object",
		"required": []string{"plan"},
		"properties": map[string]any{
			"plan": map[string]any{
				"type":        "object",
				"description": "Structured visit preparation plan with priority topics, questions to ask, pushback lines, and things to bring up if time allows",
				"required":    []string{"lead_with", "questions"},
				"properties": map[string]any{
					"lead_with": map[string]any{
						"type":        "array",
						"description": "Priority topics to lead with, sorted by urgency",
						"items": map[string]any{
							"type":     "object",
							"required": []string{"item", "urgency"},
							"properties": map[string]any{
								"item":     map[string]any{"type": "string", "description": "The topic or concern to bring up"},
								"evidence": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Supporting evidence from patient data"},
								"urgency":  map[string]any{"type": "string", "enum": []string{"critical", "high", "routine"}, "description": "How urgent this topic is"},
							},
						},
					},
					"questions": map[string]any{
						"type":        "array",
						"description": "Specific questions the patient should ask the doctor",
						"items": map[string]any{
							"type":     "object",
							"required": []string{"text", "rationale", "order_rank"},
							"properties": map[string]any{
								"text":       map[string]any{"type": "string", "description": "The question to ask"},
								"rationale":  map[string]any{"type": "string", "description": "Why this question matters"},
								"order_rank": map[string]any{"type": "integer", "description": "Priority order (1 = most important)"},
								"asked":      map[string]any{"type": "boolean", "description": "Whether asked yet (default false)"},
							},
						},
					},
					"pushback_lines": map[string]any{
						"type":        "array",
						"description": "Prepared responses if the doctor pushes back",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"trigger":  map[string]any{"type": "string", "description": "What the doctor might say"},
								"response": map[string]any{"type": "string", "description": "How to respond"},
							},
						},
					},
					"bring_up_if_time": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"doctor_summary": map[string]any{
						"type":        "string",
						"description": "Brief summary for quick reference",
					},
				},
			},
		},
	})

	return &Tool{
		Name:        "save_visit_plan",
		Description: "Save a structured preparation plan for the patient's upcoming visit. Call this after analyzing their health data to create an actionable checklist of priority topics, questions to ask, and pushback lines.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseGeneral},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var payload struct {
				Plan entities.VisitPlan `json:"plan"`
			}
			if err := json.Unmarshal(args, &payload); err != nil {
				return nil, fmt.Errorf("parse plan: %w", err)
			}

			if deps.DB == nil || uc.VisitID == "" {
				return map[string]string{"status": "saved"}, nil
			}

			planJSON, _ := json.Marshal(payload.Plan)
			_, err := deps.DB.ExecContext(ctx,
				`UPDATE visits SET plan_json = $1, updated_at = NOW() WHERE id = $2`,
				planJSON, uc.VisitID)
			if err != nil {
				return nil, fmt.Errorf("save plan: %w", err)
			}
			return map[string]string{"status": "saved"}, nil
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Saving visit preparation plan..." },
		ResultSummary: func(_ any) string { return "Visit preparation plan saved" },
	}
}
