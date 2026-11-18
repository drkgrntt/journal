package jobs

import (
	"journal/internal/logger"
	"journal/internal/models"
	"time"

	"gorm.io/gorm"
)

const requestLogPruneInterval = 6 * time.Hour

// pruneRequestLog deletes request log rows older than the retention period.
func pruneRequestLog() {
	n, err := deleteRequestLogBefore(DB(), time.Now().UTC().Add(-models.RequestLogRetention))
	if err != nil {
		logger.Error("Error pruning request log", "error", err.Error())
		return
	}
	if n > 0 {
		logger.Info("Pruned request log", "rows", n)
	}
}

// deleteRequestLogBefore hard-deletes rows created before cutoff (Unscoped, so
// nothing lingers behind a soft-delete flag) and returns how many it removed.
func deleteRequestLogBefore(db *gorm.DB, cutoff time.Time) (int64, error) {
	res := db.Unscoped().Where("created_at < ?", cutoff).Delete(&models.Analytic{})
	return res.RowsAffected, res.Error
}
