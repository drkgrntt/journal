package models

import (
	"time"
)

func init() {
	registerModel(&Journal{})
}

type Journal struct {
	*Base
	Date  *time.Time `gorm:"type:timestamptz;not null" json:"date,omitempty"`
	Entry string     `gorm:"type:text;not null" json:"entry"`

	JournalType   *JournalType `gorm:"not null" json:"journalType,omitempty"`
	JournalTypeID int          `gorm:"type:int;not null" json:"journalTypeId,omitempty"`

	Rating   *Rating `gorm:"not null" json:"rating,omitempty"`
	RatingID int     `gorm:"type:int;not null" json:"ratingId,omitempty"`

	IsEncrypted bool `gorm:"type:bool;not null" json:"isEncrypted"`
}
