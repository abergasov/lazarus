package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"lazarus/internal/entities"
)

func recordOutcomeTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":     "object",
		"required": []string{"outcome"},
		"properties": map[string]any{
			"outcome": map[string]any{
				"type":        "object",
				"description": "Structured visit outcome with diagnoses, prescriptions, instructions, action items",
			},
		},
	})

	return &Tool{
		Name:        "record_outcome",
		Description: "Record the structured outcome of a visit including diagnoses made, medications prescribed, doctor's instructions, and follow-up action items.",
		Phases:      []string{entities.PhaseCompleted},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var payload struct {
				Outcome entities.VisitOutcome `json:"outcome"`
			}
			if err := json.Unmarshal(args, &payload); err != nil {
				return nil, fmt.Errorf("parse outcome: %w", err)
			}

			if deps.DB == nil || uc.VisitID == "" {
				return map[string]string{"status": "saved"}, nil
			}

			outcomeJSON, _ := json.Marshal(payload.Outcome)
			_, err := deps.DB.ExecContext(ctx,
				`UPDATE visits SET outcome_json = $1, updated_at = NOW() WHERE id = $2`,
				outcomeJSON, uc.VisitID)
			if err != nil {
				return nil, fmt.Errorf("save outcome: %w", err)
			}
			return map[string]string{"status": "saved"}, nil
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Recording visit outcome..." },
		ResultSummary: func(_ any) string { return "Visit outcome recorded" },
	}
}
