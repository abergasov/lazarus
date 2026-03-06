package seed

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ImportInteractions seeds known drug-drug interactions.
// For a full import, parse the openFDA drug interaction dataset.
func ImportInteractions(ctx context.Context, db *sqlx.DB, openFDAFile string) error {
	// Minimal seed of critical interactions
	interactions := []struct {
		drugA, drugB, severity, description, management string
	}{
		{
			"312615", "860975",
			"moderate",
			"Aspirin may reduce the cardioprotective effect of beta-blockers when used together.",
			"Monitor blood pressure and heart rate. Consider timing of doses.",
		},
		{
			"198440", "309362",
			"moderate",
			"ACE inhibitors and calcium channel blockers may cause additive hypotension.",
			"Monitor blood pressure closely when initiating combination therapy.",
		},
	}

	for _, i := range interactions {
		_, err := db.ExecContext(ctx, `
			INSERT INTO kb_drug_interactions (drug_a_rxcui, drug_b_rxcui, severity, description, management)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT DO NOTHING
		`, i.drugA, i.drugB, i.severity, i.description, i.management)
		if err != nil {
			return fmt.Errorf("insert interaction: %w", err)
		}
	}
	return nil
}
