package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"lazarus/internal/entities"
)

func checkInteractionsTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	})

	return &Tool{
		Name:        "check_interactions",
		Description: "Check for drug-drug interactions among the patient's active medications using the knowledge base.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseCompleted},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			if deps.DB == nil || deps.KBRepo == nil {
				return []any{}, nil
			}
			userID, err := uuid.Parse(uc.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user id: %w", err)
			}

			// Get active medication RxCUIs
			type medRow struct {
				RxCUI string `db:"rxcui"`
			}
			var meds []medRow
			_ = deps.DB.SelectContext(ctx, &meds, `
				SELECT rxcui FROM medications
				WHERE user_id = $1 AND is_active = TRUE AND rxcui != ''
			`, userID)

			rxcuis := make([]string, 0, len(meds))
			for _, m := range meds {
				if m.RxCUI != "" {
					rxcuis = append(rxcuis, m.RxCUI)
				}
			}

			if len(rxcuis) < 2 {
				return []any{}, nil
			}
			return deps.KBRepo.GetDrugInteractions(ctx, rxcuis)
		},
		HumanLabel: func(_ json.RawMessage) string { return "Checking drug interactions..." },
		ResultSummary: func(r any) string {
			return fmt.Sprintf("Drug interaction check complete")
		},
	}
}
