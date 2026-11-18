package jobs

import (
	"journal/internal/models"
	"journal/internal/testutil"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func logRow(t *testing.T, tx *gorm.DB, page string, createdAt time.Time) {
	t.Helper()
	row := &models.Analytic{
		Base:   &models.Base{CreatedAt: createdAt, UpdatedAt: createdAt, CreatorID: &uuid.Nil, LastUpdaterID: &uuid.Nil},
		Method: "GET", Domain: "test", Page: page, Status: 200,
	}
	if err := tx.Create(row).Error; err != nil {
		t.Fatal(err)
	}
}

func TestDeleteRequestLogBefore(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		now := time.Now().UTC()
		cutoff := now.Add(-models.RequestLogRetention)
		logRow(t, tx, "/old", cutoff.Add(-time.Hour))
		logRow(t, tx, "/just-inside", cutoff.Add(time.Hour))
		logRow(t, tx, "/new", now)

		n, err := deleteRequestLogBefore(tx, cutoff)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("deleted %d rows, want 1", n)
		}

		var pages []string
		tx.Unscoped().Model(&models.Analytic{}).Order("page").Pluck("page", &pages)
		if len(pages) != 2 || pages[0] != "/just-inside" || pages[1] != "/new" {
			t.Fatalf("remaining pages = %v", pages)
		}
	})
}

// A soft-deleted row would still hold the data, so deletion must be real.
func TestDeleteRequestLogBeforeIsHardDelete(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		logRow(t, tx, "/old", time.Now().UTC().Add(-30*24*time.Hour))
		if _, err := deleteRequestLogBefore(tx, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		var n int64
		tx.Unscoped().Model(&models.Analytic{}).Count(&n)
		if n != 0 {
			t.Fatalf("%d rows still present (including soft-deleted)", n)
		}
	})
}

// Migration 000010 drops the identifying columns; this guards against a later
// migration or model change quietly bringing them back.
func TestRequestLogTableHasNoIdentifyingColumns(t *testing.T) {
	var cols []string
	err := testutil.DB(t).Raw(
		`SELECT column_name FROM information_schema.columns WHERE table_name = 'analytics'`,
	).Scan(&cols).Error
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, c := range cols {
		have[c] = true
	}
	for _, banned := range []string{"ip", "useragent", "query"} {
		if have[banned] {
			t.Errorf("analytics still has column %q", banned)
		}
	}
	for _, want := range []string{"status", "request_id"} {
		if !have[want] {
			t.Errorf("analytics is missing the %s column", want)
		}
	}
}
