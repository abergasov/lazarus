package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
	"lazarus/internal/entities"
)

type flagArgs struct {
	LoincCodes []string `json:"loinc_codes,omitempty"`
}

type FlagResult struct {
	Normals   []FlaggedValue `json:"normals"`
	Abnormals []FlaggedValue `json:"abnormals"`
	Criticals []FlaggedValue `json:"criticals"`
}

type FlaggedValue struct {
	LoincCode    string  `json:"loinc_code"`
	Name         string  `json:"name"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	Flag         string  `json:"flag"`
	RefLow       float64 `json:"ref_low,omitempty"`
	RefHigh      float64 `json:"ref_high,omitempty"`
	DeviationPct float64 `json:"deviation_pct"`
}

func flagAbnormalsTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"loinc_codes": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "LOINC codes to check. If empty, checks all recent labs.",
			},
		},
	})

	return &Tool{
		Name:        "flag_abnormals",
		Description: "Deterministically check recent lab values against age/sex-specific reference ranges. Returns lists of normal, abnormal, and critical values.",
		Phases:      []string{entities.PhasePreparing, entities.PhaseDuring},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var a flagArgs
			_ = json.Unmarshal(args, &a)

			if deps.DB == nil || deps.KBRepo == nil {
				return FlagResult{}, nil
			}

			userID, err := uuid.Parse(uc.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user id: %w", err)
			}

			// Get patient age/sex for reference range lookup
			sex := "ALL"
			age := 40
			if uc.PatientModel != nil {
				sex = uc.PatientModel.Demographics.Sex
				if !uc.PatientModel.Demographics.DateOfBirth.IsZero() {
					age = int(uc.PatientModel.Demographics.DateOfBirth.Year())
				}
			}

			// Query recent labs
			query := `
				SELECT lr.*, kl.long_name AS loinc_name
				FROM lab_results lr
				LEFT JOIN kb_loinc kl ON kl.code = lr.loinc_code
				WHERE lr.user_id = $1
				  AND lr.collected_at > NOW() - INTERVAL '90 days'
				ORDER BY lr.loinc_code, lr.collected_at DESC
			`
			type labRow struct {
				entities.LabResult
				LoincName string `db:"loinc_name"`
			}
			var rows []labRow
			if len(a.LoincCodes) > 0 {
				query = `
					SELECT lr.*, kl.long_name AS loinc_name
					FROM lab_results lr
					LEFT JOIN kb_loinc kl ON kl.code = lr.loinc_code
					WHERE lr.user_id = $1 AND lr.loinc_code = ANY($2)
					ORDER BY lr.loinc_code, lr.collected_at DESC
				`
				_ = deps.DB.SelectContext(ctx, &rows, query, userID, a.LoincCodes)
			} else {
				_ = deps.DB.SelectContext(ctx, &rows, query, userID)
			}

			result := FlagResult{}
			seen := map[string]bool{}
			for _, row := range rows {
				if seen[row.LoincCode] {
					continue
				}
				seen[row.LoincCode] = true

				refRange, _ := deps.KBRepo.GetReferenceRange(ctx, row.LoincCode, sex, age)
				fv := FlaggedValue{
					LoincCode: row.LoincCode,
					Name:      row.LoincName,
					Value:     row.Value,
					Unit:      row.Unit,
					Flag:      row.Flag,
				}
				if refRange != nil {
					if refRange.NormalLow != nil {
						fv.RefLow = *refRange.NormalLow
					}
					if refRange.NormalHigh != nil {
						fv.RefHigh = *refRange.NormalHigh
						fv.DeviationPct = deviationPct(row.Value, fv.RefLow, fv.RefHigh)
					}
				}

				switch row.Flag {
				case entities.FlagCriticalLow, entities.FlagCriticalHigh:
					result.Criticals = append(result.Criticals, fv)
				case entities.FlagLow, entities.FlagHigh:
					result.Abnormals = append(result.Abnormals, fv)
				default:
					result.Normals = append(result.Normals, fv)
				}
			}
			return result, nil
		},
		HumanLabel: func(_ json.RawMessage) string { return "Checking your lab values..." },
		ResultSummary: func(r any) string {
			if fr, ok := r.(FlagResult); ok {
				return fmt.Sprintf("%d abnormal, %d critical values found",
					len(fr.Abnormals), len(fr.Criticals))
			}
			return ""
		},
	}
}

func deviationPct(value, low, high float64) float64 {
	if high > 0 && value > high {
		return math.Abs((value-high)/high) * 100
	}
	if low > 0 && value < low {
		return math.Abs((low-value)/low) * 100
	}
	return 0
}
