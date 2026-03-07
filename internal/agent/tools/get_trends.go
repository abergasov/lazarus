package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"lazarus/internal/entities"
)

type getTrendsArgs struct {
	Months int `json:"months,omitempty"`
}

func getTrendsTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"months": map[string]any{"type": "integer", "description": "Number of months to look back (default 24)"},
		},
	})

	return &Tool{
		Name:        "get_trends",
		Description: "Calculate trend direction and significance for all lab values with multiple measurements. Identifies worsening or improving patterns.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseCompleted},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var a getTrendsArgs
			_ = json.Unmarshal(args, &a)
			if a.Months <= 0 {
				a.Months = 24
			}
			if deps.LabSvc == nil {
				return []any{}, nil
			}
			userID, err := uuid.Parse(uc.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user id: %w", err)
			}
			return deps.LabSvc.GetTrendsForUser(ctx, userID, a.Months)
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Analyzing your lab trends..." },
		ResultSummary: func(r any) string {
			if trends, ok := r.([]entities.TrendSummary); ok {
				sig := 0
				for _, t := range trends {
					if t.Significance == "significant" {
						sig++
					}
				}
				return fmt.Sprintf("Found %d trends (%d significant)", len(trends), sig)
			}
			return ""
		},
	}
}
