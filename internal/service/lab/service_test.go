package lab_test

import (
	"testing"
	"time"

	"lazarus/internal/entities"
	labsvc "lazarus/internal/service/lab"

	"github.com/stretchr/testify/assert"
)

func TestTrendDirection(t *testing.T) {
	svc := labsvc.NewService(nil, nil)

	points := []entities.DataPoint{
		{Value: 130, CollectedAt: time.Now().Add(-18 * 30 * 24 * time.Hour)},
		{Value: 145, CollectedAt: time.Now().Add(-9 * 30 * 24 * time.Hour)},
		{Value: 159, CollectedAt: time.Now()},
	}

	trend := svc.CalculateTrend("13457-7", "LDL", points)
	assert.Equal(t, "increasing", trend.Direction)
	assert.Greater(t, trend.PercentChange, 20.0)
	assert.Equal(t, "significant", trend.Significance)
}

func TestTrendDirection_Stable(t *testing.T) {
	svc := labsvc.NewService(nil, nil)
	points := []entities.DataPoint{
		{Value: 95, CollectedAt: time.Now().Add(-12 * 30 * 24 * time.Hour)},
		{Value: 97, CollectedAt: time.Now().Add(-6 * 30 * 24 * time.Hour)},
		{Value: 96, CollectedAt: time.Now()},
	}
	trend := svc.CalculateTrend("2339-0", "Glucose", points)
	assert.Equal(t, "stable", trend.Direction)
}
