package seed

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ImportICD10 seeds common ICD-10 conditions. For a full import,
// parse the CMS ICD-10-CM tabular XML file.
func ImportICD10(ctx context.Context, db *sqlx.DB, icd10File string) error {
	conditions := []struct {
		code, name, description, category string
		redFlags, commonLabs              []string
	}{
		{
			"E11", "Type 2 diabetes mellitus",
			"Chronic metabolic disease characterized by high blood glucose.",
			"Endocrine",
			[]string{"vision changes", "numbness in feet", "frequent infections"},
			[]string{"2339-0", "4548-4", "62238-1"},
		},
		{
			"I10", "Essential hypertension",
			"Persistently elevated arterial blood pressure without known cause.",
			"Cardiovascular",
			[]string{"severe headache", "chest pain", "shortness of breath"},
			[]string{"2951-2", "2823-3", "2160-0"},
		},
		{
			"E78.5", "Hyperlipidemia, unspecified",
			"Elevated levels of lipids in the bloodstream.",
			"Endocrine",
			[]string{"xanthomas", "corneal arcus"},
			[]string{"2093-3", "13457-7", "2085-9", "2571-8"},
		},
		{
			"N18", "Chronic kidney disease",
			"Progressive loss of kidney function over months or years.",
			"Nephrology",
			[]string{"decreased urine output", "swelling", "fatigue"},
			[]string{"2160-0", "62238-1", "2951-2", "2823-3"},
		},
		{
			"E03.9", "Hypothyroidism, unspecified",
			"Underactive thyroid gland producing insufficient thyroid hormone.",
			"Endocrine",
			[]string{"myxedema coma", "severe bradycardia"},
			[]string{"3016-3"},
		},
	}

	for _, c := range conditions {
		_, err := db.ExecContext(ctx, `
			INSERT INTO kb_conditions (icd10_code, name, description, category, red_flags, common_labs)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (icd10_code) DO NOTHING
		`, c.code, c.name, c.description, c.category, c.redFlags, c.commonLabs)
		if err != nil {
			return fmt.Errorf("insert condition %s: %w", c.code, err)
		}
	}
	return nil
}
