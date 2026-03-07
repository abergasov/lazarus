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
	// Always seed the minimal built-in set first
	if err := seedBuiltInLOINC(ctx, db); err != nil {
		return fmt.Errorf("seed built-in LOINC: %w", err)
	}

	if csvPath == "" {
		fmt.Println("  LOINC_CSV not set — using built-in minimal set only")
		return nil
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

// seedBuiltInLOINC inserts a minimal set of commonly used lab LOINC codes
// so the system works without the full LOINC CSV download.
func seedBuiltInLOINC(ctx context.Context, db *sqlx.DB) error {
	codes := []struct {
		code, longName, shortName, component, specimen, scale, unit string
	}{
		{"2339-0", "Glucose [Mass/volume] in Blood", "Glucose Bld-mCnc", "Glucose", "Bld", "Qn", "mg/dL"},
		{"4548-4", "Hemoglobin A1c/Hemoglobin.total in Blood", "HbA1c MFr Bld", "Hemoglobin A1c", "Bld", "Qn", "%"},
		{"2093-3", "Cholesterol [Mass/volume] in Serum or Plasma", "Cholest SerPl-mCnc", "Cholesterol", "Ser/Plas", "Qn", "mg/dL"},
		{"13457-7", "Cholesterol in LDL [Mass/volume] in Serum or Plasma by calculation", "LDLc SerPl Calc-mCnc", "Cholesterol.in LDL", "Ser/Plas", "Qn", "mg/dL"},
		{"2085-9", "Cholesterol in HDL [Mass/volume] in Serum or Plasma", "HDLc SerPl-mCnc", "Cholesterol.in HDL", "Ser/Plas", "Qn", "mg/dL"},
		{"2571-8", "Triglyceride [Mass/volume] in Serum or Plasma", "Trigl SerPl-mCnc", "Triglyceride", "Ser/Plas", "Qn", "mg/dL"},
		{"2160-0", "Creatinine [Mass/volume] in Serum or Plasma", "Creat SerPl-mCnc", "Creatinine", "Ser/Plas", "Qn", "mg/dL"},
		{"62238-1", "Glomerular filtration rate/1.73 sq M.predicted [Volume Rate/Area] in Serum, Plasma or Blood by Creatinine-based formula (CKD-EPI)", "GFR CKD-EPI", "Glomerular filtration rate/1.73 sq M.predicted", "Ser/Plas/Bld", "Qn", "mL/min/1.73m2"},
		{"3016-3", "Thyrotropin [Units/volume] in Serum or Plasma", "TSH SerPl-aCnc", "Thyrotropin", "Ser/Plas", "Qn", "mIU/L"},
		{"718-7", "Hemoglobin [Mass/volume] in Blood", "Hgb Bld-mCnc", "Hemoglobin", "Bld", "Qn", "g/dL"},
		{"2951-2", "Sodium [Moles/volume] in Serum or Plasma", "Sodium SerPl-sCnc", "Sodium", "Ser/Plas", "Qn", "mmol/L"},
		{"2823-3", "Potassium [Moles/volume] in Serum or Plasma", "Potassium SerPl-sCnc", "Potassium", "Ser/Plas", "Qn", "mmol/L"},
		{"6690-2", "Leukocytes [#/volume] in Blood by Automated count", "WBC # Bld Auto", "Leukocytes", "Bld", "Qn", "10*3/uL"},
		{"26515-7", "Platelets [#/volume] in Blood", "Platelet # Bld", "Platelets", "Bld", "Qn", "10*3/uL"},
		{"789-8", "Erythrocytes [#/volume] in Blood by Automated count", "RBC # Bld Auto", "Erythrocytes", "Bld", "Qn", "10*6/uL"},
		{"4544-3", "Hematocrit [Volume Fraction] of Blood by Automated count", "Hct VFr Bld Auto", "Hematocrit", "Bld", "Qn", "%"},
		{"1742-6", "Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma", "ALT SerPl-cCnc", "Alanine aminotransferase", "Ser/Plas", "Qn", "U/L"},
		{"1920-8", "Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma", "AST SerPl-cCnc", "Aspartate aminotransferase", "Ser/Plas", "Qn", "U/L"},
		{"1975-2", "Bilirubin.total [Mass/volume] in Serum or Plasma", "Bilirub SerPl-mCnc", "Bilirubin.total", "Ser/Plas", "Qn", "mg/dL"},
		{"6768-6", "Alkaline phosphatase [Enzymatic activity/volume] in Serum or Plasma", "ALP SerPl-cCnc", "Alkaline phosphatase", "Ser/Plas", "Qn", "U/L"},
		{"1751-7", "Albumin [Mass/volume] in Serum or Plasma", "Albumin SerPl-mCnc", "Albumin", "Ser/Plas", "Qn", "g/dL"},
		{"2345-7", "Glucose [Mass/volume] in Serum or Plasma", "Glucose SerPl-mCnc", "Glucose", "Ser/Plas", "Qn", "mg/dL"},
		{"3094-0", "Urea nitrogen [Mass/volume] in Serum or Plasma", "BUN SerPl-mCnc", "Urea nitrogen", "Ser/Plas", "Qn", "mg/dL"},
		{"2075-0", "Chloride [Moles/volume] in Serum or Plasma", "Chloride SerPl-sCnc", "Chloride", "Ser/Plas", "Qn", "mmol/L"},
		{"2028-9", "Carbon dioxide, total [Moles/volume] in Serum or Plasma", "CO2 SerPl-sCnc", "Carbon dioxide.total", "Ser/Plas", "Qn", "mmol/L"},
		{"17861-6", "Calcium [Mass/volume] in Serum or Plasma", "Calcium SerPl-mCnc", "Calcium", "Ser/Plas", "Qn", "mg/dL"},
		{"2777-1", "Phosphate [Mass/volume] in Serum or Plasma", "Phosphorus SerPl-mCnc", "Phosphate", "Ser/Plas", "Qn", "mg/dL"},
		{"2947-0", "Sodium [Moles/volume] in Blood", "Sodium Bld-sCnc", "Sodium", "Bld", "Qn", "mmol/L"},
		{"30239-8", "Vitamin D [Mass/volume] in Serum or Plasma", "Vit D SerPl-mCnc", "25-Hydroxyvitamin D", "Ser/Plas", "Qn", "ng/mL"},
		{"2276-4", "Ferritin [Mass/volume] in Serum or Plasma", "Ferritin SerPl-mCnc", "Ferritin", "Ser/Plas", "Qn", "ng/mL"},
		{"2498-4", "Iron [Mass/volume] in Serum or Plasma", "Iron SerPl-mCnc", "Iron", "Ser/Plas", "Qn", "ug/dL"},
		{"14749-6", "Glucose [Moles/volume] in Serum or Plasma", "Glucose SerPl Fasting-mCnc", "Glucose^fasting", "Ser/Plas", "Qn", "mg/dL"},
		{"2106-3", "Choriogonadotropin (pregnancy test) [Presence] in Urine", "bHCG Ur Ql", "Choriogonadotropin", "Urine", "Ord", ""},
		{"5811-5", "Specific gravity of Urine by Test strip", "Sp Gr Ur Strip", "Specific gravity", "Urine", "Qn", ""},
		{"5803-2", "pH of Urine by Test strip", "pH Ur Strip", "pH", "Urine", "Qn", "pH"},
		{"14959-1", "Microalbumin [Mass/volume] in Urine", "Microalb Ur-mCnc", "Albumin", "Urine", "Qn", "mg/L"},
	}

	count := 0
	for _, c := range codes {
		_, err := db.ExecContext(ctx, `
			INSERT INTO kb_loinc (code, long_name, short_name, component, specimen, scale, unit_of_measure)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (code) DO NOTHING
		`, c.code, c.longName, c.shortName, c.component, c.specimen, c.scale, c.unit)
		if err != nil {
			return fmt.Errorf("insert LOINC %s: %w", c.code, err)
		}
		count++
	}
	fmt.Printf("  seeded %d built-in LOINC codes\n", count)
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
