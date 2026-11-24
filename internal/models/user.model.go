package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func init() {
	registerModel(&User{})
}

type User struct {
	*Base
	Email     string `gorm:"uniqueIndex:idx_users_email,where:deleted_at IS NULL;not null" json:"email,omitempty"`
	FirstName string `gorm:"not null" json:"firstName,omitempty"`
	LastName  string `gorm:"not null" json:"lastName,omitempty"`
	Password  string `gorm:"not null" json:"-"`
	// TokenVersion is stamped into session and reset tokens. Bumping it ends
	// every session and pending reset link (see utils.CreateAccessToken).
	TokenVersion     int    `gorm:"not null;default:0" json:"-"`
	StripeCustomerID string `gorm:"type:text" json:"stripeCustomerId,omitempty"`
	// Appearance preferences (see appearance.go). Empty = default.
	Theme      string `gorm:"type:text;not null;default:''" json:"theme,omitempty"`
	FontFamily string `gorm:"type:text;not null;default:''" json:"fontFamily,omitempty"`

	// Subscription state; see subscription.go for how it's interpreted.
	SubscriptionStatus   string     `gorm:"type:text;not null;default:''" json:"subscriptionStatus,omitempty"`
	TrialEndsAt          *time.Time `gorm:"type:timestamptz" json:"trialEndsAt,omitempty"`
	CurrentPeriodEnd     *time.Time `gorm:"type:timestamptz" json:"currentPeriodEnd,omitempty"`
	StripeSubscriptionID string     `gorm:"type:text;not null;default:''" json:"-"`

	// Reminder email (paid). Message is stored encrypted; see user_reminderEmail.go.
	Timezone                string     `gorm:"type:text;not null;default:''" json:"-"`
	ReminderEmailEnabled    bool       `gorm:"not null;default:false" json:"-"`
	ReminderEmailTime       string     `gorm:"type:text;not null;default:'08:00'" json:"-"`
	ReminderEmailMessage    *string    `gorm:"type:text" json:"-"`
	ReminderEmailLastSentOn *time.Time `gorm:"type:date" json:"-"`
	EmailConfirmedAt        *time.Time `gorm:"type:timestamptz" json:"-"`
	// EmailConfirmationSentAt throttles resends of the confirmation email.
	EmailConfirmationSentAt *time.Time `gorm:"type:timestamptz" json:"-"`

	Journals             []*Journal             `gorm:"foreignKey:CreatorID" json:"journals,omitempty"`
	ActionItems          []*ActionItem          `gorm:"foreignKey:CreatorID" json:"actionItems,omitempty"`
	RecurringActionItems []*RecurringActionItem `gorm:"foreignKey:CreatorID" json:"recurringActionItems,omitempty"`
	Thankfuls            []*Thankful            `gorm:"foreignKey:CreatorID" json:"thankfuls,omitempty"`
	CustomJournalTypes   []*CustomJournalType   `gorm:"foreignKey:CreatorID" json:"customJournalTypes,omitempty"`
	// Features             []*Feature             `gorm:"many2many:user_features;" json:"features"`
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

// Password length limits. bcrypt only accepts up to 72 bytes (hashPassword
// would otherwise silently store an empty hash).
const (
	MinPasswordLength = 8
	MaxPasswordBytes  = 72
)

// ValidatePassword returns a message fit to show the person, or "" if the
// password is acceptable.
func ValidatePassword(password string) string {
	if len(password) < MinPasswordLength {
		return "Password must be at least 8 characters"
	}
	if len(password) > MaxPasswordBytes {
		return "Password must be at most 72 bytes"
	}
	return ""
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
	u.startTrial(time.Now().UTC())
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
