package seed

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ImportRxNorm seeds a minimal set of common drugs. For a full import,
// download RxNorm from https://www.nlm.nih.gov/research/umls/rxnorm/docs/rxnormfiles.html
// and parse RXNCONSO.RRF.
func ImportRxNorm(ctx context.Context, db *sqlx.DB, rxnormDir string) error {
	// Minimal seed of commonly prescribed drugs
	drugs := []struct {
		rxcui, name, generic string
		classes              []string
	}{
		{"1049502", "Metformin 500 MG", "metformin", []string{"biguanide"}},
		{"198440", "Lisinopril 10 MG", "lisinopril", []string{"ACE inhibitor"}},
		{"197361", "Atorvastatin 20 MG", "atorvastatin", []string{"statin"}},
		{"310798", "Omeprazole 20 MG", "omeprazole", []string{"PPI"}},
		{"309362", "Amlodipine 5 MG", "amlodipine", []string{"calcium channel blocker"}},
		{"308460", "Levothyroxine 50 MCG", "levothyroxine", []string{"thyroid hormone"}},
		{"314076", "Losartan 50 MG", "losartan", []string{"ARB"}},
		{"860975", "Metoprolol Succinate 50 MG", "metoprolol", []string{"beta blocker"}},
		{"312615", "Aspirin 81 MG", "aspirin", []string{"NSAID", "antiplatelet"}},
		{"197379", "Metformin 1000 MG", "metformin", []string{"biguanide"}},
	}

	for _, d := range drugs {
		_, err := db.ExecContext(ctx, `
			INSERT INTO kb_drugs (rxcui, name, generic_name, drug_classes)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (rxcui) DO NOTHING
		`, d.rxcui, d.name, d.generic, d.classes)
		if err != nil {
			return fmt.Errorf("insert drug %s: %w", d.rxcui, err)
		}
	}
	return nil
}
