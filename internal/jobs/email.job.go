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
) *models.Job {
	jobData, err := models.EncodeMetadata(data)
	if err != nil {
		logger.Error("Unable to encode data: ", "error", err.Error())
		return nil
	}

	job := models.Job{
		Base: &models.Base{
			CreatorID:     userId,
			LastUpdaterID: userId,
			Metadata:      jobData,
		},
		Type:        EMAIL_JOB_TYPE,
		ScheduledAt: &scheduledAt,
	}

	db.Save(&job)

	return &job
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

	err = data.getContentAndSubject(&config)
	if err != nil {
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
	case emails.GENERAL:
	default:
		return errors.New("invalid email name")
	}

	return nil
}
