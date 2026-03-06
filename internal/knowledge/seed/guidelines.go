package seed

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
)

// ImportGuidelines seeds clinical guidelines from text files in a directory.
// Each file should be named <title>.txt and contain the guideline body.
func ImportGuidelines(ctx context.Context, db *sqlx.DB, dir string) error {
	// Seed built-in minimal guidelines
	builtIn := []struct {
		title, body, source string
		relatedICD10        []string
		relatedLOINC        []string
	}{
		{
			"ADA Standards of Care: Glycemic Targets",
			"For most nonpregnant adults with T2DM, the ADA recommends an A1C goal of <7% (53 mmol/mol). " +
				"Less stringent A1C goals (<8%) may be appropriate for patients with a history of severe hypoglycemia, " +
				"limited life expectancy, or extensive comorbidities. " +
				"Fasting plasma glucose target: 80–130 mg/dL. Peak postprandial glucose <180 mg/dL.",
			"ADA 2024",
			[]string{"E11"},
			[]string{"4548-4", "2339-0"},
		},
		{
			"ACC/AHA Hypertension Guidelines: Blood Pressure Targets",
			"For most patients with hypertension, the target blood pressure is <130/80 mmHg. " +
				"Initiate pharmacological treatment at BP ≥130/80 with cardiovascular disease or " +
				"10-year ASCVD risk ≥10%. For stage 1 HTN (130-139/80-89), lifestyle modification is first-line.",
			"ACC/AHA 2017",
			[]string{"I10"},
			[]string{},
		},
		{
			"ACC/AHA Cholesterol Guidelines: LDL Targets",
			"For primary prevention, statin therapy is recommended when 10-year ASCVD risk ≥7.5%. " +
				"High-intensity statin therapy: atorvastatin 40-80mg or rosuvastatin 20-40mg targets ≥50% LDL reduction. " +
				"For established ASCVD, LDL target <70 mg/dL. Very high risk: LDL <55 mg/dL.",
			"ACC/AHA 2019",
			[]string{"E78.5", "I10"},
			[]string{"13457-7", "2093-3"},
		},
		{
			"KDIGO CKD Guidelines: Monitoring and Management",
			"Monitor eGFR and urine albumin-creatinine ratio (UACR) at least annually in CKD. " +
				"Blood pressure target <130/80 mmHg. ACE inhibitor or ARB recommended for CKD with proteinuria. " +
				"Refer to nephrology when eGFR <30 mL/min/1.73m².",
			"KDIGO 2024",
			[]string{"N18"},
			[]string{"62238-1", "2160-0"},
		},
	}

	for _, g := range builtIn {
		_, err := db.ExecContext(ctx, `
			INSERT INTO kb_guidelines (title, body, source, related_icd10, related_loinc)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT DO NOTHING
		`, g.title, g.body, g.source, g.relatedICD10, g.relatedLOINC)
		if err != nil {
			return fmt.Errorf("insert guideline %q: %w", g.title, err)
		}
	}

	// Also load from directory if it exists
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		title := strings.TrimSuffix(e.Name(), ".txt")
		_, err = db.ExecContext(ctx, `
			INSERT INTO kb_guidelines (title, body, source)
			VALUES ($1,$2,'file')
			ON CONFLICT DO NOTHING
		`, title, string(data))
		if err != nil {
			fmt.Printf("  warn: insert guideline file %s: %v\n", e.Name(), err)
		}
	}
	return nil
}
