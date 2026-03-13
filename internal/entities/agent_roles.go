package entities

type AgentRole struct {
	ProviderID AgentProvider `yaml:"provider"`
	Model      string        `yaml:"model"`
}

type RoleConfig struct {
	// PrepVisit model which will work before visit to doctor.
	// It will collect information about patient and prepare for visit, which will be used during visit and after visit.
	PrepVisit AgentRole `yaml:"prep_visit"`
	// DuringVisit model which will work during visit to doctor.
	// It will listen to doctor and patient conversation and catch important information, which will be used for recommendations after visit.
	DuringVisit AgentRole `yaml:"during_visit"`
	// AfterVisit model which will work after visit to doctor.
	// It will analyze the visit result and give recommendations for next steps.
	AfterVisit AgentRole `yaml:"after_visit"`
	// Vision is responsible for decode artifacts images into text
	Vision AgentRole `yaml:"vision"`
	// Embed is responsible for embedding text into vector, which will be used for search and retrieval.
	Embed AgentRole `yaml:"embed"`
}
