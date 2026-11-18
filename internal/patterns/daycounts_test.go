package patterns

import (
	"reflect"
	"testing"
	"time"
)

func dayAt(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func zone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestCountByLocalDay(t *testing.T) {
	tests := []struct {
		name  string
		times []string
		loc   *time.Location
		want  map[string]int
	}{
		{"empty", nil, time.UTC, map[string]int{}},
		{"nil location is UTC", []string{"2025-11-04T23:30:00Z"}, nil, map[string]int{"2025-11-04": 1}},
		{"same day grouped", []string{"2025-11-04T01:00:00Z", "2025-11-04T22:00:00Z", "2025-11-05T00:00:00Z"}, time.UTC,
			map[string]int{"2025-11-04": 2, "2025-11-05": 1}},
		// 23:30 on Nov 4 in New York (UTC-5) is 04:30 UTC on Nov 5.
		{"local evening stays on local day", []string{"2025-11-05T04:30:00Z"}, zone(t, "America/New_York"),
			map[string]int{"2025-11-04": 1}},
		// 00:30 on Nov 5 in Tokyo (UTC+9) is 15:30 UTC on Nov 4.
		{"local early morning is next day", []string{"2025-11-04T15:30:00Z"}, zone(t, "Asia/Tokyo"),
			map[string]int{"2025-11-05": 1}},
		{"same instants split by zone", []string{"2025-11-05T04:30:00Z", "2025-11-05T06:00:00Z"}, zone(t, "America/New_York"),
			map[string]int{"2025-11-04": 1, "2025-11-05": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var times []time.Time
			for _, s := range tt.times {
				times = append(times, dayAt(t, s))
			}
			if got := CountByLocalDay(times, tt.loc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeLens(t *testing.T) {
	tests := map[string]string{
		"":         LensRating,
		"rating":   LensRating,
		"wrote":    LensWrote,
		"thankful": LensThankful,
		"did":      LensDid,
		"Wrote":    LensRating,
		"bogus":    LensRating,
	}
	for in, want := range tests {
		if got := NormalizeLens(in); got != want {
			t.Errorf("NormalizeLens(%q) = %q, want %q", in, got, want)
		}
	}
}
