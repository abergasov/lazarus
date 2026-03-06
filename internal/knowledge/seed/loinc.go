package seed

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
)

// ImportLOINC bulk-inserts LOINC codes from the official CSV export.
// Download from https://loinc.org/downloads/ (requires free registration).
// Expected columns: LOINC_NUM, LONG_COMMON_NAME, SHORTNAME, COMPONENT, SYSTEM, SCALE_TYP, CLASSTYPE, EXAMPLE_UNITS
func ImportLOINC(ctx context.Context, db *sqlx.DB, csvPath string) error {
	if csvPath == "" {
		return fmt.Errorf("LOINC_CSV not set — skipping LOINC import")
	}
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open loinc csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		return err
	}
	idx := colIndex(headers)

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO kb_loinc (code, long_name, short_name, component, specimen, scale, unit_of_measure)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (code) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		_, err = stmt.ExecContext(ctx,
			col(row, idx, "LOINC_NUM"),
			col(row, idx, "LONG_COMMON_NAME"),
			col(row, idx, "SHORTNAME"),
			col(row, idx, "COMPONENT"),
			col(row, idx, "SYSTEM"),
			col(row, idx, "SCALE_TYP"),
			col(row, idx, "EXAMPLE_UNITS"),
		)
		if err != nil {
			continue
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  imported %d LOINC codes\n", count)
	return nil
}

// ImportReferenceRanges seeds a minimal set of common reference ranges.
func ImportReferenceRanges(ctx context.Context, db *sqlx.DB) error {
	ranges := []struct {
		loinc, sex, unit     string
		ageMin, ageMax       int
		nLow, nHigh          float64
		cLow, cHigh          float64
	}{
		// Glucose
		{"2339-0", "ALL", "mg/dL", 0, 999, 70, 99, 40, 500},
		// HbA1c
		{"4548-4", "ALL", "%", 0, 999, 0, 5.7, 0, 15},
		// Total Cholesterol
		{"2093-3", "ALL", "mg/dL", 0, 999, 0, 200, 0, 600},
		// LDL
		{"13457-7", "ALL", "mg/dL", 0, 999, 0, 100, 0, 400},
		// HDL male
		{"2085-9", "M", "mg/dL", 0, 999, 40, 60, 20, 100},
		// HDL female
		{"2085-9", "F", "mg/dL", 0, 999, 50, 60, 20, 100},
		// Triglycerides
		{"2571-8", "ALL", "mg/dL", 0, 999, 0, 150, 0, 1000},
		// Creatinine male
		{"2160-0", "M", "mg/dL", 18, 999, 0.74, 1.35, 0.3, 10},
		// Creatinine female
		{"2160-0", "F", "mg/dL", 18, 999, 0.59, 1.04, 0.3, 10},
		// eGFR
		{"62238-1", "ALL", "mL/min/1.73m2", 0, 999, 60, 120, 15, 150},
		// TSH
		{"3016-3", "ALL", "mIU/L", 0, 999, 0.4, 4.0, 0.1, 20},
		// Hemoglobin male
		{"718-7", "M", "g/dL", 18, 999, 13.5, 17.5, 7, 20},
		// Hemoglobin female
		{"718-7", "F", "g/dL", 18, 999, 12.0, 15.5, 7, 20},
		// Sodium
		{"2951-2", "ALL", "mmol/L", 0, 999, 136, 145, 120, 160},
		// Potassium
		{"2823-3", "ALL", "mmol/L", 0, 999, 3.5, 5.0, 2.5, 7.0},
	}

	for _, rr := range ranges {
		_, err := db.ExecContext(ctx, `
			INSERT INTO kb_reference_ranges
			  (loinc_code, sex, unit, age_min, age_max, normal_low, normal_high, critical_low, critical_high)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT DO NOTHING
		`, rr.loinc, rr.sex, rr.unit, rr.ageMin, rr.ageMax, rr.nLow, rr.nHigh, rr.cLow, rr.cHigh)
		if err != nil {
			return fmt.Errorf("insert ref range %s: %w", rr.loinc, err)
		}
	}
	return nil
}

func colIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[h] = i
	}
	return m
}

func col(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}
