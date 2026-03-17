package entities

type Role string

const (
	RoleUser    Role = "user"
	RoleAIModel Role = "assistant"
	RoleTool    Role = "tool"
)

func (r Role) String() string {
	return string(r)
}

type AgentRequest struct {
	Model        string
	SystemPrompt string
	Messages     []*AgentRequestMessage
	Tools        []*ToolDef
	MaxTokens    int
	Temperature  float64
	Images       []*AgentRequestImage
}

type ToolDef struct {
	Name        string
	Description string
	Schema      []byte // JSON Schema
}

type AgentRequestMessage struct {
	Role        Role // "user" | "assistant" | "tool"
	Content     string
	ToolCalls   []*AgentRequestToolCall // for future to load extra data if needs
	ToolResults []*AgentRequestToolResult
}

type AgentRequestImage struct {
	Data     []byte
	MimeType string
}

type AgentRequestToolCall struct {
	ID   string
	Name string
	Args []byte // raw JSON arguments
}

type AgentRequestToolResult struct {
	CallID  string
	Result  []byte // JSON
	IsError bool
}
