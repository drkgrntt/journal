package models

import (
	"go-starter/internal/logger"
	"go-starter/internal/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func (a *ActionItem) HasJournal() bool {
	return a.JournalID != uuid.Nil
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

func (a *ActionItem) EncryptText() error {
	encrypted, err := utils.Encrypt(a.Text)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	a.Text = encrypted
	a.IsEncrypted = true
	return nil
}

func (a *ActionItem) DecryptText() error {
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

func (a *ActionItem) BeforeSave(tx *gorm.DB) error {
	return a.EncryptText()
}

func (a *ActionItem) AfterSave(tx *gorm.DB) error {
	err := a.DecryptText()
	if err != nil {
		logger.Error(err.Error())
		a.IsEncrypted = false
	}
	return nil
}

func (a *ActionItem) AfterFind(tx *gorm.DB) error {
	err := a.DecryptText()
	if err != nil {
		logger.Error(err.Error())
		a.IsEncrypted = false
	}
	return nil
}
