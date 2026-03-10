package repository

import "lazarus/internal/storage/database"

type Repo struct {
	db database.DBConnector
}

var AllTables = []string{
	TableOneTimeKey,
	TableArtifactDerivatives,
	TableArtifacts,
	TableUser,
}

func InitRepo(db database.DBConnector) *Repo {
	return &Repo{db: db}
}
