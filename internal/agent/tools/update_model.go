package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

func updateModelTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":     "object",
		"required": []string{"updates"},
		"properties": map[string]any{
			"updates": map[string]any{
				"type":        "object",
				"description": "Partial patient model updates (key_concerns, open_follow_ups, conditions, communication_style, etc.)",
			},
		},
	})

	return &Tool{
		Name:        "update_model",
		Description: "Update the persistent patient model with new insights from this interaction (new concerns, follow-ups, communication preferences).",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseCompleted},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			if deps.DB == nil {
				return map[string]string{"status": "ok"}, nil
			}

			userID, err := uuid.Parse(uc.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user id: %w", err)
			}

			var updates map[string]any
			if err := json.Unmarshal(args, &updates); err != nil {
				return nil, fmt.Errorf("parse updates: %w", err)
			}
			inner, ok := updates["updates"].(map[string]any)
			if ok {
				updates = inner
			}

			modelRepo := repository.NewPatientModelRepo(deps.DB)
			model, err := modelRepo.Load(ctx, userID)
			if err != nil {
				model = &entities.PatientModel{UserID: userID}
			}

			// Apply simple string-field updates
			if v, ok := updates["communication_style"].(string); ok {
				model.CommunicationStyle = v
			}
			if v, ok := updates["medication_adherence"].(string); ok {
				model.MedAdherence = v
			}

			if err := modelRepo.Save(ctx, model); err != nil {
				return nil, fmt.Errorf("save model: %w", err)
			}
			return map[string]string{"status": "updated"}, nil
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Updating patient model..." },
		ResultSummary: func(_ any) string { return "Patient model updated" },
	}
}
