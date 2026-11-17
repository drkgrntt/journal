package jobs

import (
	"errors"
	"fmt"
	"journal/internal/emails"
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const EMAIL_JOB_TYPE = "email"

type EmailData struct {
	Name       string                   `json:"name"`
	Variables  any                      `json:"variables"`
	Recipients []*emails.EmailRecipient `json:"recipients"`
	MailgunID  string                   `json:"mailgun_id"`
}

func ScheduleEmailJob(
	userId uuid.UUID,
	data *EmailData,
	scheduledAt time.Time,
) (*models.Job, error) {
	return scheduleEmailJobTx(db, userId, data, scheduledAt)
}

// scheduleEmailJobTx is ScheduleEmailJob against a specific handle, so callers
// can create the job inside their own transaction.
func scheduleEmailJobTx(
	tx *gorm.DB,
	userId uuid.UUID,
	data *EmailData,
	scheduledAt time.Time,
) (*models.Job, error) {
	jobData, err := models.EncodeMetadata(data)
	if err != nil {
		logger.Error("Unable to encode data: ", "error", err.Error())
		return nil, err
	}

	job := models.Job{
		Base: &models.Base{
			CreatorID:     &userId,
			LastUpdaterID: &userId,
			Metadata:      jobData,
		},
		Type:        EMAIL_JOB_TYPE,
		ScheduledAt: &scheduledAt,
	}

	err = tx.Save(&job).Error
	if err != nil {
		logger.Error("Unable to save email job: ", "error", err.Error())
		return nil, err
	}

	return &job, nil
}

func sendEmail(job *models.Job) error {
	err, data := models.CastMetadata[EmailData](job.Metadata)
	if err != nil {
		return err
	}

	config := emails.EmailConfig{
		Recipients: data.Recipients,
		EmailName:  data.Name,
		Variables: []utils.KeyValue{
			{
				Key:   "jobId",
				Value: job.ID.String(),
			},
			{
				Key:   "userId",
				Value: job.CreatorID.String(),
			},
		},
	}

	if data.Name == emails.REMINDER_EMAIL {
		skip, err := prepareReminderEmail(job, &data, &config)
		if err != nil {
			return err
		}
		if skip {
			job.Notes = "Reminder email skipped (disabled or lapsed since scheduling)"
			return nil
		}
	} else if err = data.getContentAndSubject(&config); err != nil {
		return err
	}

	message, id, err := emails.SendEmail(&config)
	if err != nil {
		return err
	}

	job.Notes = fmt.Sprintf("Email sent with message: %s", message)
	data.MailgunID = id
	job.Metadata, err = models.EncodeMetadata(&data)
	if err != nil {
		return err
	}
	// Update here because .Save doesn't seem to catch that the metadata changes
	db.Model(job).Update("metadata", job.Metadata)

	return nil
}

func (data *EmailData) getContentAndSubject(config *emails.EmailConfig) error {
	switch data.Name {
	case emails.FORGOT_PASSWORD:
		err, vars := utils.CastToType[emails.ForgotPasswordVariables](data.Variables)
		if err != nil {
			return err
		}
		config.Content = emails.ForgotPassword(&vars)
		config.Subject = "Forgot Password"
	case emails.GENERAL:
		err, vars := utils.CastToType[emails.GeneralVariables](data.Variables)
		if err != nil {
			return err
		}
		config.Content = emails.General(&vars)
		config.Subject = vars.Subject
	case emails.FEEDBACK:
		err, vars := utils.CastToType[emails.FeedbackVariables](data.Variables)
		if err != nil {
			return err
		}
		config.Content = emails.Feedback(&vars)
		config.Subject = "Journal Feedback"
	default:
		return errors.New("invalid email name")
	}

	return nil
}
