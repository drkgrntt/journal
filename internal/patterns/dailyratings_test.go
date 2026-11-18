package patterns

import (
	"testing"
	"time"
)

func ratedEntry(t time.Time, v int) RatedEntry { return RatedEntry{Date: t, RatingValue: &v} }
func unratedEntry(t time.Time) RatedEntry      { return RatedEntry{Date: t} }
func ratingAt(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDailyRatings(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	tests := []struct {
		name    string
		entries []RatedEntry
		loc     *time.Location
		want    []DayRating
	}{
		{"empty", nil, time.UTC, []DayRating{}},
		{"averages same day", []RatedEntry{ratedEntry(ratingAt("2025-11-04T10:00:00Z"), 3), ratedEntry(ratingAt("2025-11-04T20:00:00Z"), 4)}, time.UTC,
			[]DayRating{{"2025-11-04", 3.5, 2}}},
		{"unrated ignored", []RatedEntry{unratedEntry(ratingAt("2025-11-04T10:00:00Z")), ratedEntry(ratingAt("2025-11-04T11:00:00Z"), 5)}, time.UTC,
			[]DayRating{{"2025-11-04", 5, 1}}},
		{"only unrated yields no day", []RatedEntry{unratedEntry(ratingAt("2025-11-04T10:00:00Z"))}, time.UTC, []DayRating{}},
		{"nil loc is UTC", []RatedEntry{ratedEntry(ratingAt("2025-11-04T23:30:00Z"), 2)}, nil,
			[]DayRating{{"2025-11-04", 2, 1}}},
		{"tz boundary moves to previous day", []RatedEntry{ratedEntry(ratingAt("2025-11-05T02:00:00Z"), 4)}, ny,
			[]DayRating{{"2025-11-04", 4, 1}}},
		{"tz boundary splits UTC day", []RatedEntry{ratedEntry(ratingAt("2025-11-05T02:00:00Z"), 1), ratedEntry(ratingAt("2025-11-05T15:00:00Z"), 5)}, ny,
			[]DayRating{{"2025-11-04", 1, 1}, {"2025-11-05", 5, 1}}},
		{"sorted by date", []RatedEntry{ratedEntry(ratingAt("2025-12-01T12:00:00Z"), 3), ratedEntry(ratingAt("2025-01-01T12:00:00Z"), 3)}, time.UTC,
			[]DayRating{{"2025-01-01", 3, 1}, {"2025-12-01", 3, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DailyRatings(tt.entries, tt.loc)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}
