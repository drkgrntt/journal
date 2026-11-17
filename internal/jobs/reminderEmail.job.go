package jobs

import (
	"errors"
	"os"
	"time"

	"journal/internal/emails"
	"journal/internal/logger"
	"journal/internal/models"

	"gorm.io/gorm"
)

var reminderEmailCheckInterval = time.Minute * time.Duration(3)

func scheduleDueReminderEmails() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Recovered from panic while scheduling reminder emails", "panic", r)
		}
	}()

	var users []*models.User
	err := db.
		Where("reminder_email_enabled = ?", true).
		Find(&users).Error
	if err != nil {
		logger.Error("Error finding reminder email users", "error", err.Error())
		return
	}

	now := time.Now()
	for _, user := range users {
		due, today := models.ReminderEmailDue(user.ReminderEmailSettings(), now)
		if !due {
			continue
		}
		if err := queueReminderEmail(user, today); err != nil {
			logger.Error("Error queueing reminder email", "error", err.Error())
		}
	}
}

// queueReminderEmail claims today for the user and creates the email job in a
// single transaction. The claim is a conditional UPDATE, so if two runners (or
// a retry/restart) race, exactly one sees RowsAffected == 1 and the other does
// nothing. If job creation fails the claim rolls back with it.
func queueReminderEmail(user *models.User, today string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Where("reminder_email_enabled = ?", true).
			Where("reminder_email_last_sent_on IS NULL OR reminder_email_last_sent_on <> ?", today).
			Update("reminder_email_last_sent_on", today)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil // already claimed
		}

		// The message is NOT put in the job (jobs are stored unencrypted); it is
		// read and decrypted at send time.
		data := &EmailData{
			Name:      emails.REMINDER_EMAIL,
			Variables: &emails.ReminderEmailVariables{RootUrl: os.Getenv("ROOT_URL")},
			Recipients: []*emails.EmailRecipient{
				{Name: user.FullName(), Email: user.Email},
			},
		}
		_, err := scheduleEmailJobTx(tx, user.ID, data, time.Now())
		return err
	})
}

// prepareReminderEmail fills in the content at send time. skip is true when
// the user has since turned the email off or lapsed.
func prepareReminderEmail(job *models.Job, data *EmailData, config *emails.EmailConfig) (skip bool, err error) {
	if job.CreatorID == nil {
		return false, errors.New("reminder email job has no user")
	}
	var user models.User
	err = db.First(&user, "id = ?", *job.CreatorID).Error
	if err != nil {
		return false, err
	}
	if !user.ReminderEmailEnabled || !user.IsSubscribed() {
		return true, nil
	}

	config.Content = emails.ReminderEmail(&emails.ReminderEmailVariables{
		RootUrl: os.Getenv("ROOT_URL"),
		Message: user.ReminderEmailMessageText(),
	})
	config.Subject = "Your reminder is waiting"
	return false, nil
}
