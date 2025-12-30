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
	Frequency   time.Duration `gorm:"type:int" json:"frequency,omitempty"`
	StartsAt    *time.Time    `gorm:"type:timestamptz" json:"startsAt,omitempty"`
	ActionItems []*ActionItem `gorm:"foreignKey:RecurringActionItemID" json:"actionItems,omitempty"`
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
		a.IsEncrypted = false
	}
	return nil
}

func (a *RecurringActionItem) AfterFind(tx *gorm.DB) error {
	err := a.DecryptText()
	if err != nil {
		logger.Error(err.Error())
		a.IsEncrypted = false
	}
	return nil
}
