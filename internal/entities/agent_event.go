package entities

type AgentEventType string

const (
	EventTypeText     AgentEventType = "text"
	EventTypeToolCall AgentEventType = "tool_call"
	EventTypeUsage    AgentEventType = "usage"
	EventTypeDone     AgentEventType = "done"
	EventTypeError    AgentEventType = "error"
)

func (a AgentEventType) String() string {
	return string(a)
}

type AgentEvent struct {
	Type         AgentEventType
	Text         string
	ToolCall     *AgentToolCall
	InputTokens  int64
	OutputTokens int64
	Error        error
}

type AgentToolCall struct {
	ID   string
	Name string
	Args []byte // raw JSON arguments
}
