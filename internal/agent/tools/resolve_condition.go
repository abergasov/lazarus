package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

func resolveConditionTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":     "object",
		"required": []string{"condition_name", "reason"},
		"properties": map[string]any{
			"condition_name": map[string]any{
				"type":        "string",
				"description": "Name of the condition to mark as resolved (must match an existing active condition)",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Why this condition is considered resolved (e.g. 'completed treatment course', 'negative follow-up test', 'patient reports symptoms resolved')",
			},
		},
	})

	return &Tool{
		Name:        "resolve_condition",
		Description: "Mark a condition as resolved in the patient's health profile. Use this when you learn that a previously active condition has been treated, a follow-up test was negative, or the patient confirms the issue is resolved. This prevents the AI from continuing to flag resolved conditions.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseCompleted, entities.PhaseGeneral},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			if deps.DB == nil {
				return map[string]string{"status": "ok"}, nil
			}

			var payload struct {
				ConditionName string `json:"condition_name"`
				Reason        string `json:"reason"`
			}
			if err := json.Unmarshal(args, &payload); err != nil {
				return nil, fmt.Errorf("parse args: %w", err)
			}

			userID, err := uuid.Parse(uc.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user id: %w", err)
			}

			modelRepo := repository.NewPatientModelRepo(deps.DB)
			model, err := modelRepo.Load(ctx, userID)
			if err != nil {
				return map[string]string{
					"status": "no_profile",
					"note":   "No patient profile found.",
				}, nil
			}

			found := false
			nameL := strings.ToLower(strings.TrimSpace(payload.ConditionName))
			for i, c := range model.ActiveConditions {
				if strings.ToLower(c.Name) == nameL || strings.Contains(strings.ToLower(c.Name), nameL) {
					model.ActiveConditions[i].Status = "resolved"
					found = true
					break
				}
			}

			if !found {
				return map[string]string{
					"status": "not_found",
					"note":   fmt.Sprintf("No active condition matching '%s' found.", payload.ConditionName),
				}, nil
			}

			// Also resolve matching key concerns
			for i, c := range model.KeyConcerns {
				if strings.Contains(strings.ToLower(c.Description), nameL) {
					model.KeyConcerns[i].Status = "resolved"
				}
			}

			if err := modelRepo.Save(ctx, model); err != nil {
				return nil, fmt.Errorf("save model: %w", err)
			}

			return map[string]string{
				"status": "resolved",
				"note":   fmt.Sprintf("'%s' marked as resolved. Reason: %s", payload.ConditionName, payload.Reason),
			}, nil
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Updating condition status..." },
		ResultSummary: func(r any) string {
			if m, ok := r.(map[string]string); ok {
				if m["status"] == "resolved" {
					return "Condition marked as resolved"
				}
				return m["note"]
			}
			return "Done"
		},
	}
}
