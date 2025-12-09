package models

import (
	"time"

	"github.com/google/uuid"
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
	AccountID   *uuid.UUID `gorm:"type:uuid;not null;"`
}
