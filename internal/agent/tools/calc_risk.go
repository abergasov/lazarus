package tools

import (
	"context"
	"encoding/json"

	"lazarus/internal/entities"
	risksvc "lazarus/internal/service/risk"
)

type calcRiskArgs struct {
	TotalCholesterol float64 `json:"total_cholesterol,omitempty"`
	HDL              float64 `json:"hdl,omitempty"`
	SystolicBP       float64 `json:"systolic_bp,omitempty"`
	OnBPMeds         bool    `json:"on_bp_meds,omitempty"`
	Diabetic         bool    `json:"diabetic,omitempty"`
	Smoker           bool    `json:"smoker,omitempty"`
}

func calcRiskTool(deps *Deps) *Tool {
	schema := mustSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"total_cholesterol": map[string]any{"type": "number"},
			"hdl":               map[string]any{"type": "number"},
			"systolic_bp":       map[string]any{"type": "number"},
			"on_bp_meds":        map[string]any{"type": "boolean"},
			"diabetic":          map[string]any{"type": "boolean"},
			"smoker":            map[string]any{"type": "boolean"},
		},
	})

	return &Tool{
		Name:        "calc_risk",
		Description: "Calculate 10-year ASCVD cardiovascular risk using ACC/AHA Pooled Cohort Equations.",
		Phases:      []string{entities.PhasePreparing},
		Schema:      schema,
		Execute: func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error) {
			var a calcRiskArgs
			_ = json.Unmarshal(args, &a)

			if deps.RiskSvc == nil || uc.PatientModel == nil {
				return map[string]any{"error": "insufficient data"}, nil
			}

			demo := uc.PatientModel.Demographics
			if demo.DateOfBirth.IsZero() {
				return map[string]any{"error": "date of birth not set"}, nil
			}

			input := risksvc.ASCVDInput{
				Age:              int(demo.DateOfBirth.Year()),
				Sex:              demo.Sex,
				TotalCholesterol: a.TotalCholesterol,
				HDL:              a.HDL,
				SystolicBP:       a.SystolicBP,
				OnBPMeds:         a.OnBPMeds,
				Diabetic:         a.Diabetic,
				Smoker:           demo.Smoker || a.Smoker,
			}
			if input.TotalCholesterol == 0 {
				input.TotalCholesterol = 200 // default
			}
			if input.HDL == 0 {
				input.HDL = 50
			}
			if input.SystolicBP == 0 {
				input.SystolicBP = 120
			}

			score := deps.RiskSvc.ASCVD10Year(input)
			return score, nil
		},
		HumanLabel:    func(_ json.RawMessage) string { return "Calculating cardiovascular risk..." },
		ResultSummary: func(r any) string {
			if score, ok := r.(*entities.RiskScore); ok {
				return "ASCVD 10-year risk: " + score.Category
			}
			return ""
		},
	}
}
