package seed

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
	"lazarus/internal/provider"
)

func EmbedTable(ctx context.Context, db *sqlx.DB, p provider.Provider, table, textExpr string) error {
	rows, err := db.QueryxContext(ctx,
		fmt.Sprintf(`SELECT ctid::text AS ctid, %s AS text FROM %s WHERE embedding IS NULL`, textExpr, table))
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		CTID string `db:"ctid"`
		Text string `db:"text"`
	}

	batch := make([]row, 0, 100)
	flush := func() error {
		for _, r := range batch {
			vec, err := p.Embed(ctx, r.Text)
			if err != nil {
				return fmt.Errorf("embed: %w", err)
			}
			_, err = db.ExecContext(ctx,
				fmt.Sprintf(`UPDATE %s SET embedding = $1 WHERE ctid = $2::tid`, table),
				pgvector.NewVector(vec), r.CTID)
			if err != nil {
				return err
			}
		}
		batch = batch[:0]
		return nil
	}

	for rows.Next() {
		var r row
		if err := rows.StructScan(&r); err != nil {
			return err
		}
		batch = append(batch, r)
		if len(batch) >= 100 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

func BuildVectorIndexes(ctx context.Context, db *sqlx.DB) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_kb_loinc_emb ON kb_loinc USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)`,
		`CREATE INDEX IF NOT EXISTS idx_kb_drugs_emb ON kb_drugs USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)`,
		`CREATE INDEX IF NOT EXISTS idx_kb_conditions_emb ON kb_conditions USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)`,
		`CREATE INDEX IF NOT EXISTS idx_kb_guidelines_emb ON kb_guidelines USING ivfflat (embedding vector_cosine_ops) WITH (lists = 50)`,
	}
	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("index: %w: %s", err, idx)
		}
	}
	return nil
}
