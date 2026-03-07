package entities

import (
	"time"

	"github.com/google/uuid"
)

type LabResult struct {
	ID             uuid.UUID  `json:"id"              db:"id"`
	UserID         uuid.UUID  `json:"user_id"         db:"user_id"`
	DocumentID     *uuid.UUID `json:"document_id"     db:"document_id"`
	LoincCode      *string    `json:"loinc_code"      db:"loinc_code"`
	Value          float64    `json:"value"           db:"value"`
	Unit           *string    `json:"unit"            db:"unit"`
	ReferenceLow   *float64   `json:"reference_low"   db:"reference_low"`
	ReferenceHigh  *float64   `json:"reference_high"  db:"reference_high"`
	Flag           string     `json:"flag"            db:"flag"`
	LabName        *string    `json:"lab_name"        db:"lab_name"`
	NormalizedName *string    `json:"normalized_name" db:"normalized_name"`
	CollectedAt    time.Time  `json:"collected_at"    db:"collected_at"`
	CreatedAt      time.Time  `json:"created_at"      db:"created_at"`
}

const (
	FlagNormal       = "normal"
	FlagLow          = "low"
	FlagHigh         = "high"
	FlagCriticalLow  = "critical_low"
	FlagCriticalHigh = "critical_high"
)

// AnnotatedLab — lab result enriched with LOINC name for display
type AnnotatedLab struct {
	LabResult
	LoincName    string  `json:"loinc_name"`
	DeviationPct float64 `json:"deviation_pct"`
}

// TrendSummary — computed trend for one LOINC code
type TrendSummary struct {
	LoincCode      string      `json:"loinc_code"`
	Name           string      `json:"name"`
	DataPoints     []DataPoint `json:"data_points"`
	Direction      string      `json:"direction"`      // "increasing" | "decreasing" | "stable"
	Slope          float64     `json:"slope"`
	PercentChange  float64     `json:"percent_change"`
	Significance   string      `json:"significance"`   // "significant" | "borderline" | "noise"
	CurrentFlag    string      `json:"current_flag"`
	Interpretation string      `json:"interpretation"`
}

type DataPoint struct {
	Value       float64   `json:"value"        db:"value"`
	CollectedAt time.Time `json:"collected_at" db:"collected_at"`
	Flag        string    `json:"flag"         db:"flag"`
}
