// Package patterns holds pure, database-free logic behind the Patterns tab, so
// it can be tested without Postgres.
package patterns

import (
	"sort"
	"time"
)

// RatedEntry is the only part of a journal entry the pattern code needs: when
// it happened and (if any) its rating's numeric value. Never entry text.
type RatedEntry struct {
	Date        time.Time
	RatingValue *int
}

// DayRating is one local day's average rating across its rated entries.
type DayRating struct {
	Date    string  `json:"date"` // YYYY-MM-DD in the user's timezone
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// DailyRatings buckets entries into days in loc (UTC if nil) and averages the
// rated ones. Unrated entries are ignored; days with none are omitted. The
// result is sorted by date.
func DailyRatings(entries []RatedEntry, loc *time.Location) []DayRating {
	if loc == nil {
		loc = time.UTC
	}
	type acc struct{ sum, count int }
	days := map[string]*acc{}
	for _, e := range entries {
		if e.RatingValue == nil {
			continue
		}
		key := e.Date.In(loc).Format("2006-01-02")
		a := days[key]
		if a == nil {
			a = &acc{}
			days[key] = a
		}
		a.sum += *e.RatingValue
		a.count++
	}
	out := make([]DayRating, 0, len(days))
	for key, a := range days {
		out = append(out, DayRating{Date: key, Average: float64(a.sum) / float64(a.count), Count: a.count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}
