package nanny_test

import (
	"testing"
	"time"

	"github.com/hunterwilliams/familyfinance/internal/domain/nanny"
	"github.com/stretchr/testify/assert"
)

func TestHoursBetween(t *testing.T) {
	start := time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	assert.Equal(t, 1.0, nanny.HoursBetween(start, end))
}

func TestCostSatang(t *testing.T) {
	// 5 hours at 200 THB/hour = 1000 THB = 100000 satang
	assert.Equal(t, int64(100000), nanny.CostSatang(5, 20000))
}
