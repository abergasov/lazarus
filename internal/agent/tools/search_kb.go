package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"lazarus/internal/entities"
)

type searchKBArgs struct {
	Query string `json:"query"`
	Type  string `json:"type,omitempty"` // "guideline" | "condition" | "all"
	TopK  int    `json:"top_k,omitempty"`
}

func searchKBTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":     "object",
		"required": []string{"query"},
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Search query for medical knowledge base"},
			"type":  map[string]any{"type": "string", "enum": []string{"guideline", "condition", "all"}, "description": "Type of knowledge to search"},
			"top_k": map[string]any{"type": "integer", "description": "Number of results (default 5)"},
		},
	})

	return &Tool{
		Name:        "search_kb",
		Description: "Search the medical knowledge base for guidelines, conditions, and clinical evidence relevant to the patient's situation.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseCompleted},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var a searchKBArgs
			_ = json.Unmarshal(args, &a)
			if a.Query == "" {
				return nil, fmt.Errorf("query is required")
			}
			if a.TopK <= 0 {
				a.TopK = 5
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
				return nil, fmt.Errorf("embed query: %w", err)
			}

			switch a.Type {
			case "guideline":
				return deps.KBRepo.SearchGuidelines(ctx, embedding, a.TopK)
			case "condition":
				return deps.KBRepo.SearchConditions(ctx, embedding, a.TopK)
			default:
				guidelines, _ := deps.KBRepo.SearchGuidelines(ctx, embedding, a.TopK/2+1)
				conditions, _ := deps.KBRepo.SearchConditions(ctx, embedding, a.TopK/2+1)
				return map[string]any{"guidelines": guidelines, "conditions": conditions}, nil
			}
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Searching medical knowledge base..." },
		ResultSummary: func(_ any) string { return "Found relevant medical guidelines and conditions" },
	}
}
