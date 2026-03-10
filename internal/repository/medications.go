package repository

import (
	"context"
	"fmt"
	"lazarus/internal/entities"
	"lazarus/internal/storage/database"
	"lazarus/internal/utils"
	"strings"
	"time"

	"github.com/google/uuid"
)

const TableMedications = "medications"

var (
	tableMedicationsCols = []string{
		"id",
		"user_id",
		"rxcui",
		"name",
		"dose",
		"frequency",
		"route",
		"prescriber",
		"is_active",
		"started_at",
		"ended_at",
		"created_at",
		"updated_at",
	}
	tableMedicationsColsStr = strings.Join(tableMedicationsCols, ",")
)

func (r *Repo) InsertMedication(ctx context.Context, medication *entities.Medication) (uuid.UUID, error) {
	id := uuid.New()
	q, p := utils.GenerateInsertSQL(TableMedications, map[string]any{
		"id":         id,
		"user_id":    medication.UserID,
		"rxcui":      medication.RxCUI,
		"name":       strip(medication.Name),
		"dose":       strip(medication.Dose),
		"frequency":  medication.Frequency,
		"route":      medication.Route,
		"prescriber": medication.Prescriber,
		"is_active":  medication.IsActive,
		"started_at": medication.StartedAt,
		"ended_at":   medication.EndedAt,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	})
	_, err := r.db.Client().ExecContext(ctx, q, p...)
	return id, err
}

func (r *Repo) GetAllMedicationsForUser(ctx context.Context, userID uuid.UUID) ([]*entities.Medication, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE user_id = $1`, tableMedicationsColsStr, TableMedications)
	return database.QueryRowsToStruct[entities.Medication](ctx, r.db.Client(), q, userID)
}
