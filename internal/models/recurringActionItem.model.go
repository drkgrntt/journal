package models

import (
	"journal/internal/logger"
	"journal/internal/utils"
	"time"

	"gorm.io/gorm"
)

func init() {
	registerModel(&RecurringActionItem{})
}

type RecurringActionItem struct {
	*Base
	Text        string        `gorm:"type:text;not null" json:"text"`
	IsEncrypted bool          `gorm:"type:bool;not null" json:"isEncrypted"`
	Frequency   time.Duration `gorm:"type:bigint" json:"frequency,omitempty"`
	StartsAt    *time.Time    `gorm:"type:timestamptz" json:"startsAt,omitempty"`
	// EmailMe sends the owner a short email each time the routine comes up.
	EmailMe     bool          `gorm:"not null;default:false" json:"emailMe"`
	ActionItems []*ActionItem `gorm:"foreignKey:RecurringActionItemID" json:"actionItems,omitempty"`
}

// DueAt reports whether a new action item should be created for this routine at
// now: it has started, and either it has never come up or its most recent action
// item was completed and neither created nor completed in the current period.
// A period is Frequency long, counted from StartsAt. ActionItems must be loaded
// newest first, as the once-a-minute cron does. It is a pure function of the
// routine and the clock, with no database access.
func (a *RecurringActionItem) DueAt(now time.Time) bool {
	if a.StartsAt == nil || a.StartsAt.After(now) {
		return false
	}
	if len(a.ActionItems) == 0 {
		return true
	}

	recent := a.ActionItems[0]
	// Still waiting on the last one.
	if recent.CompletedAt == nil {
		return false
	}

	frequency := int64(a.Frequency.Seconds())
	if frequency <= 0 {
		return false
	}
	periodsSinceStart := (now.Unix() - a.StartsAt.Unix()) / frequency
	periodStart := a.StartsAt.Add(time.Duration(periodsSinceStart) * a.Frequency)

	// Already created, or already completed, in this period.
	if recent.CreatedAt.After(periodStart) || recent.CompletedAt.After(periodStart) {
		return false
	}
	return true
}

func (a *RecurringActionItem) EncryptText() error {
	encrypted, err := utils.Encrypt(a.Text)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	a.Text = encrypted
	a.IsEncrypted = true
	return nil
}

func (a *RecurringActionItem) DecryptText() error {
	if !a.IsEncrypted {
		return nil
	}

	decrypted, err := utils.Decrypt(a.Text)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	a.Text = decrypted
	return nil
}

func (a *RecurringActionItem) BeforeSave(tx *gorm.DB) error {
	return a.EncryptText()
}

func (a *RecurringActionItem) AfterSave(tx *gorm.DB) error {
	err := a.DecryptText()
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}

func (a *RecurringActionItem) AfterFind(tx *gorm.DB) error {
	err := a.DecryptText()
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}
