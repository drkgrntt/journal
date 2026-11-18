package models

import (
	"testing"
	"time"
)

var dueNow = time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)

// routineStartedDaysAgo is a daily routine that started the given number of
// days and six hours before dueNow, so the current period began six hours ago.
func routineStartedDaysAgo(days int, items ...*ActionItem) *RecurringActionItem {
	start := dueNow.Add(-time.Duration(days)*24*time.Hour - 6*time.Hour)
	return &RecurringActionItem{StartsAt: &start, Frequency: 24 * time.Hour, ActionItems: items}
}

// item builds an action item created hoursAgo hours before dueNow and, if
// completedHoursAgo is non-negative, completed that long before dueNow.
func item(createdHoursAgo, completedHoursAgo int) *ActionItem {
	a := &ActionItem{Base: &Base{CreatedAt: dueNow.Add(-time.Duration(createdHoursAgo) * time.Hour)}}
	if completedHoursAgo >= 0 {
		c := dueNow.Add(-time.Duration(completedHoursAgo) * time.Hour)
		a.CompletedAt = &c
	}
	return a
}

func TestRecurringActionItemDueAt(t *testing.T) {
	// Daily routine; the current period began six hours before dueNow.
	future := dueNow.Add(time.Hour)
	zero := routineStartedDaysAgo(3, item(30, 28))
	zero.Frequency = 0
	negative := routineStartedDaysAgo(3, item(30, 28))
	negative.Frequency = -time.Hour

	tests := []struct {
		name string
		r    *RecurringActionItem
		want bool
	}{
		{"never started (no start time)", &RecurringActionItem{Frequency: time.Hour}, false},
		{"starts in the future", &RecurringActionItem{StartsAt: &future, Frequency: time.Hour}, false},
		{"started, never come up yet", routineStartedDaysAgo(3), true},
		{"last one still open: waits", routineStartedDaysAgo(3, item(30, -1)), false},
		{"last one done in an earlier period: due again", routineStartedDaysAgo(3, item(30, 28)), true},
		{"created in the current period: not again", routineStartedDaysAgo(3, item(2, 1)), false},
		{"completed in the current period (created before it): not again", routineStartedDaysAgo(3, item(30, 1)), false},
		{"done long ago: still just one", routineStartedDaysAgo(3, item(200, 190)), true},
		{"no frequency set but has history: skipped, not looped", zero, false},
		{"negative frequency: skipped", negative, false},
		{"only the newest item decides", routineStartedDaysAgo(3, item(30, 28), item(60, -1)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.DueAt(dueNow); got != tt.want {
				t.Fatalf("DueAt = %v, want %v", got, tt.want)
			}
		})
	}
}

// A routine with no history is due whatever its frequency; only a routine with
// history needs the frequency to work out the period.
func TestRecurringActionItemDueAtFirstTimeIgnoresFrequency(t *testing.T) {
	r := routineStartedDaysAgo(1)
	r.Frequency = 0
	if !r.DueAt(dueNow) {
		t.Fatal("a never-created routine should be due")
	}
}
