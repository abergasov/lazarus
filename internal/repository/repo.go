package repository

import (
	"lazarus/internal/storage/database"
	"strings"
)

type Repo struct {
	db database.DBConnector
}

var AllTables = []string{
	TableOneTimeKey,
	TableLabResult,
	TableMedications,
	TableArtifactDerivatives,
	TableArtifacts,
	TableUser,
}

func InitRepo(db database.DBConnector) *Repo {
	return &Repo{db: db}
}

func strip(s string) string {
	s = strings.ToLower(s)
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}
