package models

import (
	"journal/internal/logger"
	"journal/internal/utils"

	"gorm.io/gorm"
)

func init() {
	registerModel(&Reminder{})
}

type Reminder struct {
	*Base
	Text        string `gorm:"type:text;not null" json:"text"`
	IsEncrypted bool   `gorm:"type:bool;not null" json:"isEncrypted"`
}

func (r *Reminder) EncryptText() error {
	encrypted, err := utils.Encrypt(r.Text)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	r.Text = encrypted
	r.IsEncrypted = true
	return nil
}

func (r *Reminder) DecryptText() error {
	if !r.IsEncrypted {
		return nil
	}

	decrypted, err := utils.Decrypt(r.Text)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	r.Text = decrypted
	return nil
}

func (r *Reminder) BeforeSave(tx *gorm.DB) error {
	return r.EncryptText()
}

func (r *Reminder) AfterSave(tx *gorm.DB) error {
	return r.DecryptText()
}

func (r *Reminder) AfterFind(tx *gorm.DB) error {
	return r.DecryptText()
}
