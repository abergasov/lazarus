package entities

import "time"

// Phase constants
const (
	PhasePreparing = "preparing"
	PhaseDuring    = "during"
	PhaseCompleted = "completed"
)

// AgentSession — stored in DB
type AgentSession struct {
	ID               string    `json:"id"                 db:"id"`
	UserID           string    `json:"user_id"            db:"user_id"`
	VisitID          string    `json:"visit_id"           db:"visit_id"`
	Phase            string    `json:"phase"              db:"phase"`
	ProviderID       string    `json:"provider_id"        db:"provider_id"`
	ModelID          string    `json:"model_id"           db:"model_id"`
	Messages         []byte    `json:"messages"           db:"messages"`
	TokenCountInput  int       `json:"token_count_input"  db:"token_count_input"`
	TokenCountOutput int       `json:"token_count_output" db:"token_count_output"`
	CreatedAt        time.Time `json:"created_at"         db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"         db:"updated_at"`
}

// ConversationMessage — element of AgentSession.Messages
type ConversationMessage struct {
	Role      string    `json:"role"` // "user" | "assistant" | "tool"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ---- SSE Client Events ----

type ClientEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

const (
	EventAgentPlan  = "agent_plan"
	EventToolStart  = "tool_start"
	EventToolDone   = "tool_done"
	EventTextDelta  = "text_delta"
	EventStructured = "structured"
	EventDone       = "done"
	EventError      = "error"
)

type AgentPlanPayload struct {
	Steps []PlanStep `json:"steps"`
}

type PlanStep struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Tool  string `json:"tool,omitempty"`
}

type ToolStartPayload struct {
	StepID string `json:"step_id"`
	Tool   string `json:"tool"`
	Label  string `json:"label"`
}

type ToolDonePayload struct {
	StepID  string `json:"step_id"`
	Success bool   `json:"success"`
	Summary string `json:"summary,omitempty"`
}

type TextDeltaPayload struct {
	Text string `json:"text"`
}

type StructuredPayload struct {
	OutputType string `json:"output_type"` // "visit_plan" | "action_items" | "risk_summary"
	Data       any    `json:"data"`
}

type DonePayload struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
