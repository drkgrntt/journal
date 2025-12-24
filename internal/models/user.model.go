package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func init() {
	registerModel(&User{})
}

type User struct {
	*Base
	Email     string `gorm:"unique;not null" json:"email,omitempty"`
	FirstName string `gorm:"not null" json:"firstName,omitempty"`
	LastName  string `gorm:"not null" json:"lastName,omitempty"`
	Password  string `gorm:"not null" json:"-"`

	Journals []*Journal `gorm:"foreignKey:CreatorID" json:"journals,omitempty"`
}

func (u *User) FullName() string {
	fullName := ""
	if u.FirstName != "" {
		fullName += u.FirstName
	}
	if u.LastName != "" {
		if u.FirstName != "" {
			fullName += " "
		}
		fullName += u.LastName
	}
	if fullName == "" {
		fullName = u.Email
	}
	return fullName
}

func (u *User) hashPassword(password string) string {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword)
}

func (u *User) ComparePasswords(candidatePassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(candidatePassword))
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.Password = u.hashPassword(u.Password)
	return nil
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Password") {
		updatedUser := tx.Statement.Dest.(*User)
		hashedPassword := u.hashPassword(updatedUser.Password)
		tx.Statement.SetColumn("Password", hashedPassword)
	}
	return nil
}
