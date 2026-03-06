package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"lazarus/internal/entities"
)

type lookupArgs struct {
	Query string `json:"query"`
}

func lookupConditionTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":     "object",
		"required": []string{"query"},
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Condition name, ICD-10 code, or symptom to look up"},
		},
	})

	return &Tool{
		Name:        "lookup_condition",
		Description: "Look up a medical condition in the knowledge base to find description, red flags, and commonly associated lab tests.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseCompleted},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var a lookupArgs
			_ = json.Unmarshal(args, &a)
			if a.Query == "" {
				return nil, fmt.Errorf("query is required")
			}
			if deps.KBRepo == nil || deps.ProviderReg == nil {
				return []any{}, nil
			}

			embedProvider, _, err := deps.ProviderReg.ForRole("embed")
			if err != nil {
				return nil, fmt.Errorf("embed provider: %w", err)
			}
			embedding, err := embedProvider.Embed(ctx, a.Query)
			if err != nil {
				return nil, fmt.Errorf("embed: %w", err)
			}
			return deps.KBRepo.SearchConditions(ctx, embedding, 3)
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Looking up condition..." },
		ResultSummary: func(_ any) string { return "Found matching conditions" },
	}
}
