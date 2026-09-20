package jobs

import (
	gormlogger "gorm.io/gorm/logger"
	"journal/internal/database"
	"journal/internal/logger"
	"journal/internal/models"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	db               *gorm.DB
	scheduleDuration = time.Second * time.Duration(10)
)

func init() {
	if db == nil {
		db = database.New().DB
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		logger.Error("Error starting scheduler", "error", err.Error())
		return
	}

	// Create the next time
	_, err = s.NewJob(
		gocron.DurationJob(scheduleDuration),
		gocron.NewTask(runJobs),
	)
	if err != nil {
		logger.Error("Error starting scheduler", "error", err.Error())
	}

	s.Start()
}

var (
	maxRetries         = 10
	timeBetweenRetries = time.Minute * time.Duration(5)
	jobLimit           = 100
)

func runJobs() {
	// logger.Info("Running jobs")

	var jobs []*models.Job
	now := time.Now()

	// Fetch and claim due jobs inside a transaction with a row lock so that
	// multiple job-runner instances can't grab and double-process the same
	// job: SKIP LOCKED lets a concurrent runner skip rows another runner is
	// already claiming, and marking AttemptedAt before committing keeps it
	// from matching the "due" query again once the lock is released.
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.
			Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Error)}).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("processed_at IS NULL").
			Where("scheduled_at <= ?", now).
			Where("retries <= ? OR retries IS NULL", maxRetries).
			Where("attempted_at <= ? OR attempted_at IS NULL", now.Add(-timeBetweenRetries)).
			Order("priority ASC").
			Order("scheduled_at ASC").
			Limit(jobLimit).
			Find(&jobs).Error
		if err != nil {
			return err
		}

		for _, job := range jobs {
			if job.AttemptedAt != nil {
				job.Retries += 1
			} else {
				job.Retries = 0
			}
			job.AttemptedAt = &now
		}

		if len(jobs) > 0 {
			return tx.Save(&jobs).Error
		}

		return nil
	})

	if err != nil {
		logger.Error("Error finding jobs", "error", err.Error())
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Recovered from panic while running jobs", "panic", r)
			if len(jobs) > 0 {
				db.Save(&jobs)
			}
		}
	}()

	for _, job := range jobs {
		switch strings.ToLower(job.Type) {
		case EMAIL_JOB_TYPE:
			err = sendEmail(job)
		default:
			logger.Warn("No job process found with the corresponding type", "type", job.Type)
			continue
		}

		if err == nil {
			processedAt := time.Now()
			job.ProcessedAt = &processedAt
		} else {
			if job.Notes != "" {
				job.Notes += "\n"
			}
			job.Notes += err.Error()
		}
	}

	if len(jobs) > 0 {
		err = db.Save(&jobs).Error
		if err != nil {
			logger.Error("Error saving processed jobs", "error", err.Error())
		}
	}

	// logger.Info("Finished running jobs")
}
