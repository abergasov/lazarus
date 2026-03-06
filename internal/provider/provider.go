package provider

import "context"

type Provider interface {
	ID() string
	Stream(ctx context.Context, req *Request) (<-chan Event, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Request struct {
	Model       string
	System      string
	Messages    []Message
	Tools       []ToolDef
	MaxTokens   int
	Temperature float64
	Images      []Image
}

type Message struct {
	Role        string       // "user" | "assistant" | "tool"
	Content     string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
}

type Image struct {
	Data     []byte
	MimeType string
}

type ToolDef struct {
	Name        string
	Description string
	Schema      []byte // JSON Schema
}

type ToolCall struct {
	ID   string
	Name string
	Args []byte // raw JSON arguments
}

type ToolResult struct {
	CallID  string
	Result  []byte // JSON
	IsError bool
}

type Event struct {
	Type         EventType
	Text         string
	ToolCall     *ToolCall
	InputTokens  int
	OutputTokens int
	Error        error
}

type EventType string

const (
	EventTypeText     EventType = "text"
	EventTypeToolCall EventType = "tool_call"
	EventTypeUsage    EventType = "usage"
	EventTypeDone     EventType = "done"
	EventTypeError    EventType = "error"
)
