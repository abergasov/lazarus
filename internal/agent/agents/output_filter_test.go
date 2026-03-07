package agents

import (
	"strings"
	"testing"
)

func TestFilterDangerousDirectives_WithSafetyQualifier(t *testing.T) {
	// Should NOT be modified — has safety qualifier nearby
	input := "Your potassium is high. Ask your doctor about whether you should stop taking lisinopril."
	result := filterUnsafeOutput(input)
	if strings.Contains(result, "Discuss with your doctor: ") {
		t.Errorf("should not modify text with safety qualifier, got: %s", result)
	}
}

func TestFilterDangerousDirectives_WithoutSafetyQualifier(t *testing.T) {
	// Should be modified — no safety qualifier
	input := "Your potassium is dangerously high. Stop taking lisinopril immediately."
	result := filterUnsafeOutput(input)
	if !strings.Contains(result, "Discuss with your doctor") {
		t.Errorf("should add safety framing, got: %s", result)
	}
}

func TestFilterDosageInstruction(t *testing.T) {
	input := "You should take 500 mg of metformin twice daily."
	result := filterUnsafeOutput(input)
	if !strings.Contains(result, "Discuss with your doctor") {
		t.Errorf("should flag dosage instruction, got: %s", result)
	}
}

func TestFilterDosageWithQualifier(t *testing.T) {
	input := "Your doctor may suggest you take 500 mg of metformin. Discuss with your doctor before making any changes."
	result := filterUnsafeOutput(input)
	if strings.Contains(result, "Discuss with your doctor:**") {
		t.Errorf("should not double-flag when qualifier present, got: %s", result)
	}
}

func TestAppendDisclaimer_WithMedicalRecommendation(t *testing.T) {
	input := "Based on your labs, I recommend you starting a statin therapy."
	result := filterUnsafeOutput(input)
	if !strings.Contains(result, "not medical advice") {
		t.Errorf("should append disclaimer for medical recommendation, got: %s", result)
	}
}

func TestAppendDisclaimer_AlreadyPresent(t *testing.T) {
	input := "I recommend you starting a statin therapy. This is not medical advice."
	result := filterUnsafeOutput(input)
	// Should not double-add
	if strings.Count(result, "not medical advice") > 1 {
		t.Errorf("should not double-add disclaimer, got: %s", result)
	}
}

func TestNoFilter_SafeText(t *testing.T) {
	input := "Your hemoglobin A1c is 5.4%, which is within the normal range of 4.0-5.6%. This means your blood sugar control over the past 3 months has been good."
	result := filterUnsafeOutput(input)
	if result != input {
		t.Errorf("should not modify safe informational text, got: %s", result)
	}
}
