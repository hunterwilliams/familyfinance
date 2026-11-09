package nanny

import "time"

func HoursBetween(start, end time.Time) float64 {
	d := end.Sub(start)
	if d <= 0 {
		return 0
	}
	return d.Hours()
}

func CostSatang(hours float64, hourlyRateSatang int64) int64 {
	return int64(hours * float64(hourlyRateSatang))
}
