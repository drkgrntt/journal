package patterns

import (
	"testing"
	"time"
)

func weekRated(t time.Time, v int) WeekByTimeEntry { return WeekByTimeEntry{Date: t, RatingValue: &v} }
func weekUnrated(t time.Time) WeekByTimeEntry      { return WeekByTimeEntry{Date: t} }
func utc(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.UTC)
}

// 2024-06-02 is a Sunday.
func TestComputeWeekByTime(t *testing.T) {
	plus7 := time.FixedZone("plus7", 7*3600)
	cases := []struct {
		name        string
		entries     []WeekByTimeEntry
		loc         *time.Location
		granularity string
		row, col    int
		wantCount   int
		wantAvg     float64
	}{
		{"tz crosses into Sunday morning", []WeekByTimeEntry{weekRated(utc(2024, 6, 1, 23, 30), 4)}, plus7, "period", 0, 1, 1, 4},
		{"same instant in UTC is Saturday evening", []WeekByTimeEntry{weekRated(utc(2024, 6, 1, 23, 30), 4)}, nil, "period", 6, 3, 1, 4},
		{"04:59 is night", []WeekByTimeEntry{weekRated(utc(2024, 6, 2, 4, 59), 2)}, time.UTC, "period", 0, 0, 1, 2},
		{"05:00 is morning", []WeekByTimeEntry{weekRated(utc(2024, 6, 2, 5, 0), 2)}, time.UTC, "period", 0, 1, 1, 2},
		{"16:59 is afternoon", []WeekByTimeEntry{weekRated(utc(2024, 6, 2, 16, 59), 3)}, time.UTC, "period", 0, 2, 1, 3},
		{"17:00 is evening", []WeekByTimeEntry{weekRated(utc(2024, 6, 2, 17, 0), 3)}, time.UTC, "period", 0, 3, 1, 3},
		{"3hour 02:59 is the 00:00 block", []WeekByTimeEntry{weekRated(utc(2024, 6, 2, 2, 59), 1)}, time.UTC, "3hour", 0, 0, 1, 1},
		{"3hour 03:00 is the 03:00 block", []WeekByTimeEntry{weekRated(utc(2024, 6, 2, 3, 0), 5)}, time.UTC, "3hour", 0, 1, 1, 5},
		{"averages within a cell", []WeekByTimeEntry{weekRated(utc(2024, 6, 2, 18, 0), 3), weekRated(utc(2024, 6, 9, 19, 0), 4)}, time.UTC, "period", 0, 3, 2, 3.5},
		{"unrated ignored", []WeekByTimeEntry{weekUnrated(utc(2024, 6, 2, 18, 0)), weekRated(utc(2024, 6, 2, 18, 5), 2)}, time.UTC, "period", 0, 3, 1, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeWeekByTime(tc.entries, tc.loc, tc.granularity)
			cell := got.Grid[tc.row][tc.col]
			if cell.Count != tc.wantCount || cell.Average != tc.wantAvg {
				t.Fatalf("cell = %+v, want count %d avg %v", cell, tc.wantCount, tc.wantAvg)
			}
			if got.RatedCount() != tc.wantCount {
				t.Fatalf("RatedCount = %d, want %d", got.RatedCount(), tc.wantCount)
			}
		})
	}
}

func TestComputeWeekByTimeShape(t *testing.T) {
	for gran, cols := range map[string]int{"period": 4, "3hour": 8, "": 4, "bogus": 4} {
		got := ComputeWeekByTime(nil, nil, gran)
		if len(got.Columns) != cols || len(got.Grid) != 7 || len(got.Rows) != 7 || got.Rows[0] != "Sunday" {
			t.Fatalf("%q: bad shape %+v", gran, got)
		}
		for _, row := range got.Grid {
			if len(row) != cols {
				t.Fatalf("%q: row has %d cells, want %d", gran, len(row), cols)
			}
		}
		if got.RatedCount() != 0 {
			t.Fatalf("%q: empty input should have no rated entries", gran)
		}
	}
	if got := ComputeWeekByTime(nil, nil, "3hour"); got.Columns[0] != "00:00" || got.Columns[7] != "21:00" {
		t.Fatalf("3hour labels wrong: %v", got.Columns)
	}
}
