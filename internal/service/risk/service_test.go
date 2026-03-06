package risk_test

import (
	"testing"

	risksvc "lazarus/internal/service/risk"

	"github.com/stretchr/testify/assert"
)

func TestASCVD_HighRisk(t *testing.T) {
	svc := risksvc.NewService()

	// 65-year-old male smoker with elevated cholesterol and BP → clearly high risk (≥20%)
	input := risksvc.ASCVDInput{
		Age:              65,
		Sex:              "M",
		TotalCholesterol: 240,
		HDL:              40,
		SystolicBP:       150,
		OnBPMeds:         false,
		Diabetic:         false,
		Smoker:           true,
	}

	score := svc.ASCVD10Year(input)
	assert.Greater(t, score.Value, 20.0)
	assert.Equal(t, "high", score.Category)
	assert.True(t, score.ActionNeeded) // > 7.5% threshold
}

func TestASCVD_LowRisk(t *testing.T) {
	svc := risksvc.NewService()
	input := risksvc.ASCVDInput{
		Age: 40, Sex: "F", TotalCholesterol: 170, HDL: 65,
		SystolicBP: 110, OnBPMeds: false, Diabetic: false, Smoker: false,
	}
	score := svc.ASCVD10Year(input)
	assert.Less(t, score.Value, 5.0)
	assert.Equal(t, "low", score.Category)
	assert.False(t, score.ActionNeeded)
}
