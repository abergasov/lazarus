//go:build ignore

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"lazarus/internal/config"
	"lazarus/internal/knowledge/seed"
	"lazarus/internal/provider"
	"lazarus/internal/storage/database"
)

func main() {
	cfgPath := flag.String("config", "configs/app_conf.yml", "path to config")
	flag.Parse()

	cfg, err := config.InitConf(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	db, err := database.InitDBConnect(ctx, &cfg.ConfigDB, "")
	if err != nil {
		log.Fatal(err)
	}

	reg, err := provider.NewRegistry(&cfg.LLM)
	if err != nil {
		log.Fatal(err)
	}
	embedProvider, _, err := reg.ForRole("embed")
	if err != nil {
		log.Fatal(err)
	}

	sqlDB := db.Client()

	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"LOINC codes", func(c context.Context) error { return seed.ImportLOINC(c, sqlDB, os.Getenv("LOINC_CSV")) }},
		{"Reference ranges", func(c context.Context) error { return seed.ImportReferenceRanges(c, sqlDB) }},
		{"RxNorm drugs", func(c context.Context) error { return seed.ImportRxNorm(c, sqlDB, os.Getenv("RXNORM_DIR")) }},
		{"Drug interactions", func(c context.Context) error { return seed.ImportInteractions(c, sqlDB, os.Getenv("OPENFDA_FILE")) }},
		{"ICD-10 conditions", func(c context.Context) error { return seed.ImportICD10(c, sqlDB, os.Getenv("ICD10_FILE")) }},
		{"Guidelines", func(c context.Context) error { return seed.ImportGuidelines(c, sqlDB, "migrations/seed/guidelines/") }},
		{"Embed LOINC", func(c context.Context) error { return seed.EmbedTable(c, sqlDB, embedProvider, "kb_loinc", "long_name") }},
		{"Embed drugs", func(c context.Context) error { return seed.EmbedTable(c, sqlDB, embedProvider, "kb_drugs", "name") }},
		{"Embed conditions", func(c context.Context) error {
			return seed.EmbedTable(c, sqlDB, embedProvider, "kb_conditions", "name || ' ' || COALESCE(description,'')")
		}},
		{"Embed guidelines", func(c context.Context) error {
			return seed.EmbedTable(c, sqlDB, embedProvider, "kb_guidelines", "title || ' ' || LEFT(body, 1000)")
		}},
		{"Build vector indexes", func(c context.Context) error { return seed.BuildVectorIndexes(c, sqlDB) }},
	}

	for _, step := range steps {
		fmt.Printf("→ %s...\n", step.name)
		if err := step.fn(ctx); err != nil {
			log.Fatalf("FAILED %s: %v", step.name, err)
		}
		fmt.Printf("  ✓ %s done\n", step.name)
	}

	fmt.Println("\nKB seeding complete.")
}
