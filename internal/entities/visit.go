package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Visit struct {
	ID           uuid.UUID  `json:"id"            db:"id"`
	UserID       uuid.UUID  `json:"user_id"       db:"user_id"`
	DoctorName   *string    `json:"doctor_name"   db:"doctor_name"`
	Specialty    *string    `json:"specialty"     db:"specialty"`
	ClinicName   *string    `json:"clinic_name"   db:"clinic_name"`
	VisitDate    *time.Time `json:"visit_date"    db:"visit_date"`
	VisitType    *string    `json:"visit_type"    db:"visit_type"`
	Reason       *string    `json:"reason"        db:"reason"`
	Status       string     `json:"status"        db:"status"`
	PlanJSON     NullJSON   `json:"plan"          db:"plan_json"`
	OutcomeJSON  NullJSON   `json:"outcome"       db:"outcome_json"`
	FollowUpDate *time.Time `json:"follow_up_date" db:"follow_up_date"`
	CreatedAt    time.Time  `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"    db:"updated_at"`
}

// NullJSON wraps json.RawMessage to handle SQL NULL scanning.
type NullJSON json.RawMessage

func (n *NullJSON) Scan(value any) error {
	if value == nil {
		*n = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*n = append((*n)[:0], v...)
	case string:
		*n = NullJSON(v)
	}
	return nil
}

func (n NullJSON) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte("null"), nil
	}
	return json.RawMessage(n).MarshalJSON()
}

const (
	VisitStatusPreparing = "preparing"
	VisitStatusDuring    = "during"
	VisitStatusCompleted = "completed"
	VisitStatusCancelled = "cancelled"
)

type VisitPlan struct {
	LeadWith      []VisitPriority `json:"lead_with"`
	Questions     []VisitQuestion `json:"questions"`
	Pushbacks     []PushbackLine  `json:"pushback_lines"`
	BringUpIfTime []string        `json:"bring_up_if_time"`
	DoctorSummary string          `json:"doctor_summary"`
	GeneratedAt   time.Time       `json:"generated_at"`
}

type VisitPriority struct {
	Item     string   `json:"item"`
	Evidence []string `json:"evidence"`
	Urgency  string   `json:"urgency"` // "critical" | "high" | "routine"
}

type VisitQuestion struct {
	Text      string `json:"text"`
	Rationale string `json:"rationale"`
	OrderRank int    `json:"order_rank"`
	Asked     bool   `json:"asked"`
}

type PushbackLine struct {
	Trigger  string `json:"trigger"`
	Response string `json:"response"`
}

type VisitOutcome struct {
	DoctorSaid    string         `json:"doctor_said"`
	Diagnoses     []Diagnosis    `json:"diagnoses"`
	Prescribed    []Prescription `json:"prescribed"`
	Instructions  []string       `json:"instructions"`
	FollowUpDate  *time.Time     `json:"follow_up_date,omitempty"`
	Gaps          []VisitGap     `json:"gaps"`
	ActionItems   []ActionItem   `json:"action_items"`
	OpenFollowUps []FollowUp     `json:"open_follow_ups"`
	RecordedAt    time.Time      `json:"recorded_at"`
}

type Diagnosis struct {
	ICD10Code string `json:"icd10_code"`
	Name      string `json:"name"`
	Status    string `json:"status"` // "confirmed" | "suspected" | "ruled_out"
}

type Prescription struct {
	Name      string `json:"name"`
	RxCUI     string `json:"rxcui,omitempty"`
	Dose      string `json:"dose"`
	Frequency string `json:"frequency"`
}

type VisitGap struct {
	Description string `json:"description"`
	Guideline   string `json:"guideline,omitempty"`
	Severity    string `json:"severity"` // "clinical" | "informational"
}

type ActionItem struct {
	Action  string     `json:"action"`
	Reason  string     `json:"reason"`
	DueDate *time.Time `json:"due_date,omitempty"`
	Done    bool       `json:"done"`
}

type FollowUp struct {
	Action    string     `json:"action"`
	Reason    string     `json:"reason"`
	FromVisit uuid.UUID  `json:"from_visit"`
	DueDate   *time.Time `json:"due_date,omitempty"`
	Completed bool       `json:"completed"`
}
