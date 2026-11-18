// Package patterns holds pure data-shaping helpers for the Patterns tab. It
// must not import the database or Fiber, so it can be tested without Postgres.
package patterns

import "time"

// Calendar lenses: what the calendar pattern shows for each day.
const (
	LensRating   = "rating"
	LensWrote    = "wrote"
	LensThankful = "thankful"
	LensDid      = "did"
)

// DayKeyLayout is the key format used by CountByLocalDay.
const DayKeyLayout = "2006-01-02"

// NormalizeLens returns lens if it is a known lens, otherwise LensRating.
func NormalizeLens(lens string) string {
	switch lens {
	case LensRating, LensWrote, LensThankful, LensDid:
		return lens
	}
	return LensRating
}

// CountByLocalDay counts times per calendar day in loc, keyed "2006-01-02".
// A nil loc is treated as UTC.
func CountByLocalDay(times []time.Time, loc *time.Location) map[string]int {
	if loc == nil {
		loc = time.UTC
	}
	counts := map[string]int{}
	for _, t := range times {
		counts[t.In(loc).Format(DayKeyLayout)]++
	}
	return counts
}
