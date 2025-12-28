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
	ProcessedAt *time.Time `gorm:"type:time"`
	ScheduledAt *time.Time `gorm:"type:time"`
	AttemptedAt *time.Time `gorm:"type:time"`
	Retries     int        `gorm:"type:int; default:0;"`
	Priority    int        `gorm:"type:int; default:10;"`
}
