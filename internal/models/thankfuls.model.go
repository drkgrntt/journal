package models

import (
	"go-starter/internal/logger"
	"go-starter/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() {
	registerModel(&Thankful{})
}

type Thankful struct {
	*Base
	Text string `gorm:"type:text;not null" json:"text"`

	Journal   *Journal  `json:"journal,omitempty"`
	JournalID uuid.UUID `gorm:"type:int" json:"journalId,omitempty"`

	IsEncrypted bool `gorm:"type:bool;not null" json:"isEncrypted"`
}

func (t *Thankful) HasJournal() bool {
	return t.JournalID != uuid.Nil
}

func (t *Thankful) HasJournalType() bool {
	if t.Journal == nil {
		return false
	}
	if t.Journal.JournalType == nil {
		return false
	}
	return true
}

func (t *Thankful) EncryptText() error {
	encrypted, err := utils.Encrypt(t.Text)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	t.Text = encrypted
	t.IsEncrypted = true
	return nil
}

func (t *Thankful) DecryptText() error {
	if !t.IsEncrypted {
		return nil
	}

	decrypted, err := utils.Decrypt(t.Text)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	t.Text = decrypted
	return nil
}

func (t *Thankful) BeforeSave(tx *gorm.DB) error {
	return t.EncryptText()
}

func (t *Thankful) AfterSave(tx *gorm.DB) error {
	err := t.DecryptText()
	if err != nil {
		logger.Error(err.Error())
		t.IsEncrypted = false
	}
	return nil
}

func (t *Thankful) AfterFind(tx *gorm.DB) error {
	err := t.DecryptText()
	if err != nil {
		logger.Error(err.Error())
		t.IsEncrypted = false
	}
	return nil
}
