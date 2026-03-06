package knowledge

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

type KBHit struct {
	ID       string  `db:"id"`
	Title    string  `db:"title"`
	Excerpt  string  `db:"excerpt"`
	Source   string  `db:"source"`
	Score    float64 `db:"score"`
	Category string  `db:"category"`
}

func (r *Repository) SearchGuidelines(ctx context.Context, embedding []float32, topK int) ([]KBHit, error) {
	vec := pgvector.NewVector(embedding)
	query := `
		SELECT id::text, title, LEFT(body, 500) AS excerpt, COALESCE(source,'') AS source,
		       1 - (embedding <=> $1) AS score, '' AS category
		FROM kb_guidelines
		ORDER BY embedding <=> $1
		LIMIT $2
	`
	var hits []KBHit
	if err := r.db.SelectContext(ctx, &hits, query, vec, topK); err != nil {
		return nil, fmt.Errorf("search guidelines: %w", err)
	}
	return hits, nil
}

func (r *Repository) SearchConditions(ctx context.Context, embedding []float32, topK int) ([]KBHit, error) {
	vec := pgvector.NewVector(embedding)
	query := `
		SELECT icd10_code AS id, name AS title,
		       LEFT(COALESCE(description,''), 500) AS excerpt, COALESCE(category,'') AS category,
		       1 - (embedding <=> $1) AS score, COALESCE(category,'') AS source
		FROM kb_conditions
		ORDER BY embedding <=> $1
		LIMIT $2
	`
	var hits []KBHit
	return hits, r.db.SelectContext(ctx, &hits, query, vec, topK)
}

func (r *Repository) GetReferenceRange(ctx context.Context, loincCode string, sex string, age int) (*ReferenceRange, error) {
	query := `
		SELECT * FROM kb_reference_ranges
		WHERE loinc_code = $1
		  AND (sex = 'ALL' OR sex = $2)
		  AND ($3 BETWEEN COALESCE(age_min, 0) AND COALESCE(age_max, 999))
		ORDER BY sex DESC
		LIMIT 1
	`
	var rr ReferenceRange
	if err := r.db.GetContext(ctx, &rr, query, loincCode, sex, age); err != nil {
		return nil, err
	}
	return &rr, nil
}

func (r *Repository) GetDrugInteractions(ctx context.Context, rxcuis []string) ([]DrugInteraction, error) {
	if len(rxcuis) == 0 {
		return nil, nil
	}
	query := `
		SELECT di.drug_a_rxcui, di.drug_b_rxcui, a.name AS drug_a_name, b.name AS drug_b_name,
		       di.severity, COALESCE(di.description,'') AS description, COALESCE(di.management,'') AS management
		FROM kb_drug_interactions di
		JOIN kb_drugs a ON a.rxcui = di.drug_a_rxcui
		JOIN kb_drugs b ON b.rxcui = di.drug_b_rxcui
		WHERE di.drug_a_rxcui = ANY($1) AND di.drug_b_rxcui = ANY($1)
		ORDER BY
			CASE di.severity
				WHEN 'contraindicated' THEN 1
				WHEN 'major' THEN 2
				WHEN 'moderate' THEN 3
				ELSE 4
			END
	`
	var interactions []DrugInteraction
	return interactions, r.db.SelectContext(ctx, &interactions, query, rxcuis)
}

func (r *Repository) GuidelineCount(ctx context.Context) (int, error) {
	var count int
	return count, r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM kb_guidelines")
}

func (r *Repository) GetLoincName(ctx context.Context, loincCode string) (string, error) {
	var name string
	err := r.db.GetContext(ctx, &name, "SELECT long_name FROM kb_loinc WHERE code = $1", loincCode)
	return name, err
}

type ReferenceRange struct {
	ID           int      `db:"id"`
	LoincCode    string   `db:"loinc_code"`
	AgeMin       *int     `db:"age_min"`
	AgeMax       *int     `db:"age_max"`
	Sex          string   `db:"sex"`
	NormalLow    *float64 `db:"normal_low"`
	NormalHigh   *float64 `db:"normal_high"`
	AbnormalLow  *float64 `db:"abnormal_low"`
	AbnormalHigh *float64 `db:"abnormal_high"`
	CriticalLow  *float64 `db:"critical_low"`
	CriticalHigh *float64 `db:"critical_high"`
	Unit         string   `db:"unit"`
}

type DrugInteraction struct {
	DrugARxCUI  string `db:"drug_a_rxcui"`
	DrugBRxCUI  string `db:"drug_b_rxcui"`
	DrugAName   string `db:"drug_a_name"`
	DrugBName   string `db:"drug_b_name"`
	Severity    string `db:"severity"`
	Description string `db:"description"`
	Management  string `db:"management"`
}
