package aws

import (
	"math"
	"time"

	"github.com/c3xdev/c3x/internal/usagesync"
)

// daysPerMonth is what a monthly_* catalog quantity means. A measured
// total over the window is scaled to it, so the default 30-day run is
// the total itself. Days the window covers but CloudWatch returned no
// datapoint for count as zero: an omitted count is not traffic.
const daysPerMonth = 30

func round4(v float64) float64 {
	return math.Round(v*1e4) / 1e4
}

// monthlyCount is the window total expressed per 30 days, plus the same
// rate for each calendar month the points fall in. The month's denominator
// is how much of that month the window covers, not how many datapoints
// came back, so a quiet day stays zero instead of being scaled away.
func monthlyCount(points []datapoint, w usagesync.Window) (float64, map[string]float64) {
	if len(points) == 0 {
		return 0, nil
	}
	var total float64
	monthSum := map[string]float64{}
	for _, p := range points {
		total += p.value
		monthSum[p.at.UTC().Format("2006-01")] += p.value
	}
	series := make(map[string]float64, len(monthSum))
	for month, sum := range monthSum {
		covered := overlapDays(month, w)
		if covered <= 0 {
			continue
		}
		series[month] = round4(sum * (daysPerMonth / covered))
	}
	return perMonth(total, w), series
}

func perMonth(total float64, w usagesync.Window) float64 {
	days := w.End.Sub(w.Start).Hours() / 24
	if days <= 0 {
		return 0
	}
	return round4(total * (daysPerMonth / days))
}

// overlapDays is the length of the window inside one calendar month.
func overlapDays(month string, w usagesync.Window) float64 {
	start, err := time.ParseInLocation("2006-01", month, time.UTC)
	if err != nil {
		return 0
	}
	end := start.AddDate(0, 1, 0)
	lo, hi := w.Start.UTC(), w.End.UTC()
	if start.After(lo) {
		lo = start
	}
	if end.Before(hi) {
		hi = end
	}
	if !hi.After(lo) {
		return 0
	}
	return hi.Sub(lo).Hours() / 24
}
