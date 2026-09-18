package models

import (
	"time"
)

func init() {
	registerModel(&Job{})
}

type Job struct {
	*Base
	Type        string     `gorm:"type:varchar(255)"`
	Notes       string     `gorm:"type:text"`
	ProcessedAt *time.Time `gorm:"type:timestamptz"`
	ScheduledAt *time.Time `gorm:"type:timestamptz"`
	AttemptedAt *time.Time `gorm:"type:timestamptz"`
	Retries     int        `gorm:"type:int; default:0;"`
	Priority    int        `gorm:"type:int; default:10;"`
}
