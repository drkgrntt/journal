package models

import (
	"go-starter/internal/logger"
	"go-starter/internal/utils"
	"time"

	"gorm.io/gorm"
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

	ActionItems []*ActionItem `gorm:"foreignKey:JournalID" json:"actionItems,omitempty"`

	IsEncrypted bool `gorm:"type:bool;not null" json:"isEncrypted"`
}

func (j *Journal) EncryptEntry() error {
	if j.IsEncrypted {
		return nil
	}

	encrypted, err := utils.Encrypt(j.Entry)
	if err != nil {
		return err
	}

	j.Entry = encrypted
	j.IsEncrypted = true
	return nil
}

func (j *Journal) DecryptEntry() error {
	if !j.IsEncrypted {
		return nil
	}

	decrypted, err := utils.Decrypt(j.Entry)
	if err != nil {
		return err
	}

	j.Entry = decrypted
	return nil
}

func (j *Journal) BeforeSave(tx *gorm.DB) error {
	return j.EncryptEntry()
}

func (j *Journal) AfterSave(tx *gorm.DB) error {
	err := j.DecryptEntry()
	if err != nil {
		logger.Error(err.Error())
		j.IsEncrypted = false
	}
	return nil
}

func (j *Journal) AfterFind(tx *gorm.DB) error {
	err := j.DecryptEntry()
	if err != nil {
		logger.Error(err.Error())
		j.IsEncrypted = false
	}
	return nil
}
