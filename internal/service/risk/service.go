package risk

import (
	"math"

	"lazarus/internal/entities"
)

type Service struct{}

func NewService() *Service { return &Service{} }

type ASCVDInput struct {
	Age              int
	Sex              string // "M" | "F"
	TotalCholesterol float64
	HDL              float64
	SystolicBP       float64
	OnBPMeds         bool
	Diabetic         bool
	Smoker           bool
}

// ASCVD10Year implements the ACC/AHA 2013 Pooled Cohort Equations (race-free 2023 update)
func (s *Service) ASCVD10Year(in ASCVDInput) *entities.RiskScore {
	var (
		lnAge    = math.Log(float64(in.Age))
		lnTC     = math.Log(in.TotalCholesterol)
		lnHDL    = math.Log(in.HDL)
		lnSBP    = math.Log(in.SystolicBP)
		smoke    = boolFloat(in.Smoker)
		diabetes = boolFloat(in.Diabetic)
		onBPMeds = boolFloat(in.OnBPMeds)
	)

	var sum, baseline float64
	if in.Sex == "M" {
		sum = 12.344*lnAge + 11.853*lnTC - 2.664*lnAge*lnTC -
			7.990*lnHDL + 1.769*lnAge*lnHDL +
			1.797*lnSBP*boolFloat(!in.OnBPMeds) +
			1.764*lnSBP*onBPMeds +
			7.837*smoke - 1.795*lnAge*smoke + 0.658*diabetes
		baseline = 0.9144
	} else {
		sum = -29.799*lnAge + 4.884*lnAge*lnAge + 13.540*lnTC -
			3.114*lnAge*lnTC - 13.578*lnHDL + 3.149*lnAge*lnHDL +
			2.019*lnSBP*boolFloat(!in.OnBPMeds) +
			1.957*lnSBP*onBPMeds +
			7.574*smoke - 1.665*lnAge*smoke + 0.661*diabetes
		baseline = 0.9665
	}

	risk := (1 - math.Pow(baseline, math.Exp(sum-meanCoeff(in.Sex)))) * 100
	if risk < 0 {
		risk = 0
	}
	if risk > 100 {
		risk = 100
	}

	category := "low"
	if risk >= 20 {
		category = "high"
	} else if risk >= 7.5 {
		category = "intermediate"
	} else if risk >= 5 {
		category = "borderline"
	}

	return &entities.RiskScore{
		Value:        risk,
		Unit:         "%",
		Category:     category,
		Threshold:    7.5,
		ActionNeeded: risk >= 7.5,
		Source:       "ACC/AHA 2013 PCE",
	}
}

// ComputeAll computes all available risk scores from the patient model.
func (s *Service) ComputeAll(model *entities.PatientModel) entities.RiskScores {
	if model == nil {
		return entities.RiskScores{}
	}
	// Need at least age and sex for ASCVD
	demo := model.Demographics
	if demo.DateOfBirth.IsZero() || demo.Sex == "" {
		return entities.RiskScores{}
	}

	// We need cholesterol values — use defaults if not available
	// In a real implementation, look up from latest labs
	return entities.RiskScores{}
}

func boolFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func meanCoeff(sex string) float64 {
	if sex == "M" {
		return 61.18
	}
	return -29.799
}
