package jobs

import (
	"go-starter/internal/database"
	"go-starter/internal/logger"
	"go-starter/internal/models"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
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

	// s.Start()
}

var (
	maxRetries         = 10
	timeBetweenRetries = time.Minute * time.Duration(5)
	jobLimit           = 100
)

func runJobs() {
	logger.Info("Running jobs")

	var jobs []*models.Job
	now := time.Now()
	err := db.
		Where("processed_at IS NULL").
		Where("scheduled_at <= ?", now).
		Where("retries <= ? OR retries IS NULL", maxRetries).
		Where("attempted_at >= ? OR attempted_at IS NULL", now.Add(timeBetweenRetries)).
		Order("priority ASC").
		Order("scheduled_at ASC").
		Limit(jobLimit).
		Find(&jobs).Error

	if err != nil {
		logger.Error("Error finding jobs", "error", err.Error())
		return
	}

	defer func() {
		if r := recover(); r != nil {
			if len(jobs) > 0 {
				db.Save(&jobs)
			}
		}
	}()

	for _, job := range jobs {
		if job.AttemptedAt != nil {
			job.Retries += 1
		} else {
			job.Retries = 0
		}

		now = time.Now()
		job.AttemptedAt = &now

		switch strings.ToLower(job.Type) {
		default:
			logger.Warn("No job process found with the corresponding type", "type", job.Type)
			continue
		}

		if err == nil {
			job.ProcessedAt = &now
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

	logger.Info("Finished running jobs")
}
