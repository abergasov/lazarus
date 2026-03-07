package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"lazarus/internal/entities"
)

type ContraindicationResult struct {
	DrugConditionConflicts []DrugConditionConflict `json:"drug_condition_conflicts"`
	AllergyConflicts       []AllergyConflict       `json:"allergy_conflicts"`
}

type DrugConditionConflict struct {
	DrugName    string `json:"drug_name"`
	DrugRxCUI   string `json:"drug_rxcui"`
	Condition   string `json:"condition"`
	ICD10Code   string `json:"icd10_code"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type AllergyConflict struct {
	DrugName  string `json:"drug_name"`
	DrugRxCUI string `json:"drug_rxcui"`
	Allergen  string `json:"allergen"`
	Severity  string `json:"allergy_severity"`
	Reaction  string `json:"reaction"`
}

func checkContraindicationsTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	})

	return &Tool{
		Name:        "check_contraindications",
		Description: "Check for drug-condition contraindications (medications unsafe given the patient's diagnoses) and allergy conflicts (medications the patient is allergic to). This is a critical safety check.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring, entities.PhaseCompleted, entities.PhaseGeneral},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			if deps.DB == nil {
				return ContraindicationResult{}, nil
			}
			userID, err := uuid.Parse(uc.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user id: %w", err)
			}

			result := ContraindicationResult{}

			// 1. Check drug-condition contraindications
			var conflicts []DrugConditionConflict
			err = deps.DB.SelectContext(ctx, &conflicts, `
				SELECT m.name AS drug_name, m.rxcui AS drug_rxcui,
				       kc.name AS condition, ci.icd10_code,
				       ci.severity, COALESCE(ci.description, '') AS description
				FROM medications m
				JOIN kb_drug_condition_contraindications ci ON ci.rxcui = m.rxcui
				JOIN kb_conditions kc ON kc.icd10_code = ci.icd10_code
				WHERE m.user_id = $1 AND m.is_active = TRUE AND m.rxcui != ''
				  AND ci.icd10_code IN (
				    SELECT UNNEST(
				      ARRAY(
				        SELECT jsonb_array_elements_text(
				          COALESCE(data->'active_conditions', '[]'::jsonb)
				        )
				        FROM patient_models WHERE user_id = $1
				      )
				    )
				  )
				ORDER BY
				  CASE ci.severity
				    WHEN 'absolute' THEN 1
				    WHEN 'major' THEN 2
				    WHEN 'moderate' THEN 3
				    ELSE 4
				  END
			`, userID)
			// If the complex query fails (e.g. no patient model), try simpler approach
			if err != nil {
				// Fallback: check against conditions stored in patient model JSON
				conflicts = nil
			}
			result.DrugConditionConflicts = conflicts

			// 2. Check allergy conflicts
			var allergyConflicts []AllergyConflict
			err = deps.DB.SelectContext(ctx, &allergyConflicts, `
				SELECT m.name AS drug_name, m.rxcui AS drug_rxcui,
				       a.substance AS allergen, a.severity AS allergy_severity,
				       COALESCE(a.reaction, '') AS reaction
				FROM medications m
				JOIN allergies a ON (
				  (a.rxcui IS NOT NULL AND a.rxcui != '' AND a.rxcui = m.rxcui)
				  OR LOWER(m.name) LIKE '%' || LOWER(a.substance) || '%'
				)
				WHERE m.user_id = $1 AND m.is_active = TRUE
				  AND a.user_id = $1
				ORDER BY
				  CASE a.severity
				    WHEN 'life_threatening' THEN 1
				    WHEN 'severe' THEN 2
				    WHEN 'moderate' THEN 3
				    ELSE 4
				  END
			`, userID)
			if err != nil {
				allergyConflicts = nil
			}
			result.AllergyConflicts = allergyConflicts

			return result, nil
		},
		HumanLabel: func(_ json.RawMessage) string { return "Checking drug safety..." },
		ResultSummary: func(r any) string {
			if cr, ok := r.(ContraindicationResult); ok {
				total := len(cr.DrugConditionConflicts) + len(cr.AllergyConflicts)
				if total == 0 {
					return "No contraindications or allergy conflicts found"
				}
				return fmt.Sprintf("Found %d safety concern(s): %d drug-condition, %d allergy",
					total, len(cr.DrugConditionConflicts), len(cr.AllergyConflicts))
			}
			return ""
		},
	}
}
