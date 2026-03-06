package lab

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
	"lazarus/internal/knowledge"
)

type Service struct {
	db     *sqlx.DB
	kbRepo *knowledge.Repository
}

func NewService(db *sqlx.DB, kbRepo *knowledge.Repository) *Service {
	return &Service{db: db, kbRepo: kbRepo}
}

func (s *Service) GetTrendsForUser(ctx context.Context, userID uuid.UUID, months int) ([]entities.TrendSummary, error) {
	type loincRow struct {
		LoincCode string `db:"loinc_code"`
		Name      string `db:"long_name"`
	}
	var loincs []loincRow
	err := s.db.SelectContext(ctx, &loincs, `
		SELECT lr.loinc_code, kl.long_name
		FROM lab_results lr
		JOIN kb_loinc kl ON kl.code = lr.loinc_code
		WHERE lr.user_id = $1
		  AND lr.collected_at > NOW() - ($2 || ' months')::interval
		GROUP BY lr.loinc_code, kl.long_name
		HAVING COUNT(*) >= 2
	`, userID, months)
	if err != nil {
		return nil, err
	}

	summaries := make([]entities.TrendSummary, 0, len(loincs))
	for _, l := range loincs {
		var pts []entities.DataPoint
		err := s.db.SelectContext(ctx, &pts, `
			SELECT value, collected_at, flag FROM lab_results
			WHERE user_id = $1 AND loinc_code = $2
			ORDER BY collected_at ASC
		`, userID, l.LoincCode)
		if err != nil {
			continue
		}
		summaries = append(summaries, s.CalculateTrend(l.LoincCode, l.Name, pts))
	}
	return summaries, nil
}

func (s *Service) GetAnnotatedLabs(ctx context.Context, userID uuid.UUID, days int) ([]entities.AnnotatedLab, error) {
	type row struct {
		entities.LabResult
		LoincName string `db:"long_name"`
	}
	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT lr.*, kl.long_name
		FROM lab_results lr
		LEFT JOIN kb_loinc kl ON kl.code = lr.loinc_code
		WHERE lr.user_id = $1
		  AND lr.collected_at > NOW() - ($2 || ' days')::interval
		ORDER BY lr.collected_at DESC
	`, userID, days)
	if err != nil {
		return nil, err
	}

	result := make([]entities.AnnotatedLab, len(rows))
	for i, r := range rows {
		result[i] = entities.AnnotatedLab{
			LabResult: r.LabResult,
			LoincName: r.LoincName,
		}
		if r.ReferenceLow != nil && r.ReferenceHigh != nil {
			result[i].DeviationPct = deviationPct(r.Value, *r.ReferenceLow, *r.ReferenceHigh)
		}
	}
	return result, nil
}

func (s *Service) CalculateTrend(loincCode, name string, points []entities.DataPoint) entities.TrendSummary {
	if len(points) < 2 {
		return entities.TrendSummary{LoincCode: loincCode, Name: name, DataPoints: points, Direction: "stable"}
	}

	n := float64(len(points))
	var sumX, sumY, sumXY, sumXX float64
	base := points[0].CollectedAt.Unix()
	for _, p := range points {
		x := float64(p.CollectedAt.Unix() - base)
		y := p.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	first := points[0].Value
	last := points[len(points)-1].Value
	pctChange := ((last - first) / first) * 100

	direction := "stable"
	if slope > 0 && math.Abs(pctChange) > 5 {
		direction = "increasing"
	} else if slope < 0 && math.Abs(pctChange) > 5 {
		direction = "decreasing"
	}

	significance := "noise"
	if math.Abs(pctChange) > 15 {
		significance = "significant"
	} else if math.Abs(pctChange) > 7 {
		significance = "borderline"
	}

	interpretation := fmt.Sprintf("%s has %s %.1f%% over the past data",
		name, direction, math.Abs(pctChange))

	return entities.TrendSummary{
		LoincCode:      loincCode,
		Name:           name,
		DataPoints:     points,
		Direction:      direction,
		Slope:          slope,
		PercentChange:  pctChange,
		Significance:   significance,
		CurrentFlag:    points[len(points)-1].Flag,
		Interpretation: interpretation,
	}
}

func deviationPct(value, low, high float64) float64 {
	if value > high {
		return math.Abs((value-high)/high) * 100
	}
	if value < low {
		return math.Abs((low-value)/low) * 100
	}
	return 0
}
