package models

import (
	"time"

	"github.com/google/uuid"
)

func init() {
	registerModel(&ActionItem{})
}

type ActionItem struct {
	*Base
	Text        string     `gorm:"type:text;not null" json:"text"`
	CompletedAt *time.Time `gorm:"type:timestamptz" json:"completedAt,omitempty"`

	Journal   *Journal  `json:"journal,omitempty"`
	JournalID uuid.UUID `gorm:"type:int" json:"journalId,omitempty"`

	IsEncrypted bool `gorm:"type:bool;not null" json:"isEncrypted"`
}

func (a *ActionItem) IsComplete() bool {
	if a.CompletedAt == nil {
		return false
	}

	if a.CompletedAt.IsZero() {
		return false
	}

	return true
}
