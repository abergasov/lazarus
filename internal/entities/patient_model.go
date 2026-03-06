package entities

import (
	"time"

	"github.com/google/uuid"
)

type PatientModel struct {
	UserID  uuid.UUID `json:"user_id"`
	Version int       `json:"version"`

	Demographics       Demographics    `json:"demographics"`
	RiskScores         RiskScores      `json:"risk_scores"`
	ActiveConditions   []Condition     `json:"active_conditions"`
	KeyConcerns        []Concern       `json:"key_concerns"`
	OpenFollowUps      []FollowUp      `json:"open_follow_ups"`
	DoctorDynamics     []DoctorDynamic `json:"doctor_dynamics"`
	CommunicationStyle string          `json:"communication_style"`
	MedAdherence       string          `json:"medication_adherence"`

	VisitCount           int       `json:"visit_count"`
	LastSynthesized      time.Time `json:"last_synthesized"`
	OnboardingCompleted  bool      `json:"onboarding_completed"`
}

type Demographics struct {
	DateOfBirth   time.Time `json:"date_of_birth"`
	Sex           string    `json:"sex"` // "M" | "F"
	HeightCM      float64   `json:"height_cm"`
	WeightKG      float64   `json:"weight_kg"`
	Smoker        bool      `json:"smoker"`
	FamilyHistory []string  `json:"family_history"`
}

type RiskScores struct {
	ASCVD10Year  *RiskScore `json:"ascvd_10yr,omitempty"`
	Framingham   *RiskScore `json:"framingham,omitempty"`
	CKDEPI       *RiskScore `json:"ckd_epi,omitempty"`
	DiabetesRisk *RiskScore `json:"diabetes_risk,omitempty"`
}

type RiskScore struct {
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	Category     string  `json:"category"` // "low" | "borderline" | "intermediate" | "high"
	Threshold    float64 `json:"threshold"`
	ActionNeeded bool    `json:"action_needed"`
	Source       string  `json:"source"`
}

type Condition struct {
	ICD10Code   string    `json:"icd10_code"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // "active" | "resolved"
	DiagnosedAt time.Time `json:"diagnosed_at"`
}

type Concern struct {
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // "critical" | "high" | "medium" | "watch"
	Evidence    []string  `json:"evidence"`
	FirstSeen   time.Time `json:"first_seen"`
	Status      string    `json:"status"` // "active" | "monitoring" | "resolved"
}

type DoctorDynamic struct {
	DoctorName string `json:"doctor_name"`
	Specialty  string `json:"specialty"`
	Style      string `json:"style"`
	Notes      string `json:"notes"`
	VisitCount int    `json:"visit_count"`
}
