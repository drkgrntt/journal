package models

import (
	"errors"
	"strings"
	"unicode/utf8"

	"journal/internal/logger"
	"journal/internal/utils"

	"gorm.io/gorm"
)

func init() {
	registerModel(&EntryTemplate{})
}

// EntryTemplate is a user-authored, reusable starting point for a new journal
// entry. Name and Body are both encrypted at rest, like Reminder.Text.
type EntryTemplate struct {
	*Base
	Name        string `gorm:"type:text;not null" json:"name"`
	Body        string `gorm:"type:text;not null" json:"body"`
	IsEncrypted bool   `gorm:"type:bool;not null" json:"isEncrypted"`
}

func (t *EntryTemplate) Encrypt() error {
	name, err := utils.Encrypt(t.Name)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	body, err := utils.Encrypt(t.Body)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	t.Name = name
	t.Body = body
	t.IsEncrypted = true
	return nil
}

func (t *EntryTemplate) Decrypt() error {
	if !t.IsEncrypted {
		return nil
	}

	name, err := utils.Decrypt(t.Name)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	body, err := utils.Decrypt(t.Body)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	t.Name = name
	t.Body = body
	return nil
}

func (t *EntryTemplate) BeforeSave(tx *gorm.DB) error {
	return t.Encrypt()
}

func (t *EntryTemplate) AfterSave(tx *gorm.DB) error {
	return t.Decrypt()
}

func (t *EntryTemplate) AfterFind(tx *gorm.DB) error {
	return t.Decrypt()
}

const (
	EntryTemplateMaxNameLength = 100
	EntryTemplateMaxBodyLength = 5000
)

// CleanEntryTemplateInput trims and validates a template's name and body. The
// body keeps its leading structure but loses trailing whitespace.
func CleanEntryTemplateInput(name, body string) (string, string, error) {
	name = strings.TrimSpace(name)
	body = strings.TrimRight(body, " \t\r\n")

	if name == "" {
		return "", "", errors.New("name is required")
	}
	if strings.TrimSpace(body) == "" {
		return "", "", errors.New("body is required")
	}
	if utf8.RuneCountInString(name) > EntryTemplateMaxNameLength {
		return "", "", errors.New("name is too long")
	}
	if utf8.RuneCountInString(body) > EntryTemplateMaxBodyLength {
		return "", "", errors.New("body is too long")
	}
	return name, body, nil
}
