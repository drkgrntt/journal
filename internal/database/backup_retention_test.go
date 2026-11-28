package database

import (
	"slices"
	"testing"
	"time"
)

// backupAt names a backup taken the given time before now, the way backup() does.
func backupAt(now time.Time, years, months, days int) backupObject {
	t := now.AddDate(-years, -months, -days)
	return backupObject{key: backupPrefix + "backup-" + t.Format("2006-01-02-15-04-05") + ".sql", t: t}
}

// The privacy page promises backups are gone within a year, so even the last
// one of a month is deleted once it is that old.
func TestBackupsOlderThanAYearArePruned(t *testing.T) {
	now := time.Date(2026, time.November, 28, 20, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		backup backupObject
		pruned bool
	}{
		{"yesterday", backupAt(now, 0, 0, 1), false},
		{"eight months", backupAt(now, 0, 8, 0), false},
		{"a day short of a year", backupAt(now, 1, 0, -1), false},
		{"exactly a year", backupAt(now, 1, 0, 0), true},
		{"three years", backupAt(now, 3, 0, 0), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Alone in its week and month, so thinning can't be what removes it.
			got := slices.Contains(backupsToPrune([]backupObject{c.backup}, now), c.backup.key)
			if got != c.pruned {
				t.Errorf("pruned = %v, want %v", got, c.pruned)
			}
		})
	}
}
