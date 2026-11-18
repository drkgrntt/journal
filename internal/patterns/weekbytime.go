// Package patterns holds pure (no DB, no Fiber) logic behind the dashboard's
// Patterns tab, so it can be tested without a live Postgres.
package patterns

import (
	"fmt"
	"time"
)

// WeekByTimeEntry is the minimum a journal entry needs to contribute to the
// week-by-time grid: when it was written and, if rated, its rating value.
// No entry text ever enters this package.
type WeekByTimeEntry struct {
	Date        time.Time
	RatingValue *int
}

// WeekByTimeCell is one day-of-week x time-block cell. Count is 0 for a cell
// with no rated entries, in which case Average is 0 and should be ignored.
type WeekByTimeCell struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// WeekByTime is the grid: Columns are the ordered time-block labels and Grid
// is 7 rows (Sunday first) by len(Columns) cells. Rows lists the day names.
type WeekByTime struct {
	Columns []string           `json:"columns"`
	Rows    []string           `json:"rows"`
	Grid    [][]WeekByTimeCell `json:"grid"`
}

var weekdayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

var periodLabels = []string{"Night", "Morning", "Afternoon", "Evening"}

// blockIndex maps an hour to its column under the granularity ("period" or
// "3hour"; anything else is treated as "period"). The rules mirror
// the former time-of-day chart (Night/Morning/Afternoon/Evening, or 3-hour blocks).
func blockIndex(hour int, granularity string) int {
	if granularity == "3hour" {
		return hour / 3
	}
	switch {
	case hour < 5:
		return 0
	case hour < 12:
		return 1
	case hour < 17:
		return 2
	default:
		return 3
	}
}

func blockLabels(granularity string) []string {
	if granularity == "3hour" {
		labels := make([]string, 8)
		for i := range labels {
			labels[i] = fmt.Sprintf("%02d:00", i*3)
		}
		return labels
	}
	return append([]string(nil), periodLabels...)
}

// ComputeWeekByTime buckets rated entries by weekday and time block in loc
// (nil means UTC) and averages the ratings in each cell. Unrated entries and
// entries with a nil date are ignored.
func ComputeWeekByTime(entries []WeekByTimeEntry, loc *time.Location, granularity string) WeekByTime {
	if loc == nil {
		loc = time.UTC
	}
	columns := blockLabels(granularity)
	sums := make([][]int, 7)
	grid := make([][]WeekByTimeCell, 7)
	for i := range grid {
		sums[i] = make([]int, len(columns))
		grid[i] = make([]WeekByTimeCell, len(columns))
	}

	for _, e := range entries {
		if e.RatingValue == nil {
			continue
		}
		local := e.Date.In(loc)
		row := int(local.Weekday())
		col := blockIndex(local.Hour(), granularity)
		sums[row][col] += *e.RatingValue
		grid[row][col].Count++
	}
	for r := range grid {
		for c := range grid[r] {
			if n := grid[r][c].Count; n > 0 {
				grid[r][c].Average = float64(sums[r][c]) / float64(n)
			}
		}
	}
	return WeekByTime{Columns: columns, Rows: append([]string(nil), weekdayNames...), Grid: grid}
}

// RatedCount is the total number of rated entries in the grid; 0 means the
// window is empty.
func (w WeekByTime) RatedCount() int {
	total := 0
	for _, row := range w.Grid {
		for _, cell := range row {
			total += cell.Count
		}
	}
	return total
}
