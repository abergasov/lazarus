package entities

type AgentRole string

const (
	AgentRolePrepVisit   AgentRole = "prep_visit"
	AgentRoleDuringVisit AgentRole = "during_visit"
	AgentRoleAfterVisit  AgentRole = "after_visit"
	AgentRoleVision      AgentRole = "vision"

	// AgentRoleEmbed for put data in RAG database, not used for now
	// maybe in future we will need it, so we can add it later
	AgentRoleEmbed AgentRole = "embed"
)

func (ag AgentRole) String() string {
	return string(ag)
}

type AgentRoleConfig struct {
	ProviderID AgentProvider `yaml:"provider"`
	Model      string        `yaml:"model"`
}

type RoleConfig struct {
	// PrepVisit model which will work before visit to doctor.
	// It will collect information about patient and prepare for visit, which will be used during visit and after visit.
	PrepVisit *AgentRoleConfig `yaml:"prep_visit"`
	// DuringVisit model which will work during visit to doctor.
	// It will listen to doctor and patient conversation and catch important information, which will be used for recommendations after visit.
	DuringVisit *AgentRoleConfig `yaml:"during_visit"`
	// AfterVisit model which will work after visit to doctor.
	// It will analyze the visit result and give recommendations for next steps.
	AfterVisit *AgentRoleConfig `yaml:"after_visit"`
	// Vision is responsible for decode artifacts images into text
	Vision *AgentRoleConfig `yaml:"vision"`
	// Embed is responsible for embedding text into vector, which will be used for search and retrieval.
	Embed *AgentRoleConfig `yaml:"embed"`
}
