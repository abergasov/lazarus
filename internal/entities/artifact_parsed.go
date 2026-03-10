package entities

import "encoding/json"

type ArtifactParsedData struct {
	LabResults  []ArtifactParsedLab       `json:"lab_results"`
	Medications []ArtifactParsedMed       `json:"medications"`
	Diagnoses   []ArtifactParsedDiagnosis `json:"diagnoses"`
	Date        string                    `json:"date"`
	Category    string                    `json:"category"`
	Specialty   string                    `json:"specialty"`
	Summary     string                    `json:"summary"`
}

type ArtifactParsedLab struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
	Unit  string          `json:"unit"`
	Range string          `json:"range"`
	Flag  string          `json:"flag"`
	Date  string          `json:"date"`
}

type ArtifactParsedMed struct {
	Name      string `json:"name"`
	Dose      string `json:"dose"`
	Frequency string `json:"frequency"`
}

type ArtifactParsedDiagnosis struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
