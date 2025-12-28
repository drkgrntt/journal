package emails

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"journal/internal/utils"
	"os"
	"time"

	"github.com/a-h/templ"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mailgun/mailgun-go/v4"
)

var defaultSender = "Contour Journal <noreply@contourjournal.com>"

type EmailRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type EmailConfig struct {
	Recipients    []*EmailRecipient
	Subject       string
	Sender        string
	WithoutLayout bool
	Content       templ.Component
	EmailName     string
	Variables     []utils.KeyValue
}

func (ec *EmailConfig) Valiate() error {
	if len(ec.Recipients) == 0 {
		return errors.New("at least one recipient must be present in EmailConfig")
	}

	for _, recipient := range ec.Recipients {
		if recipient.Name == "" || recipient.Email == "" {
			return errors.New("all recipients require a name and email")
		}
	}

	if ec.Subject == "" {
		return errors.New("subject must be present in EmailConfig")
	}

	if ec.Content == nil {
		return errors.New("a templ component must be set as your content in EmailConfig")
	}

	return nil
}

func SendEmail(emailConfig *EmailConfig) (mes string, id string, err error) {
	if err = emailConfig.Valiate(); err != nil {
		return
	}

	environment := os.Getenv("APP_ENV")

	if environment != "production" {
		emailConfig.Recipients = []*EmailRecipient{
			{
				Name:  "Derek Garnett",
				Email: "derekgarnett@hey.com",
			},
		}
	}
	mgDomain := os.Getenv("MG_DOMAIN")
	mgKey := os.Getenv("MG_KEY")

	mg := mailgun.NewMailgun(mgDomain, mgKey)
	sender := emailConfig.Sender
	if sender == "" {
		sender = defaultSender
	}
	subject := emailConfig.Subject

	var buffer bytes.Buffer
	if emailConfig.WithoutLayout {
		emailConfig.Content.Render(context.Background(), &buffer)
	} else {
		renderContext := templ.WithChildren(context.Background(), emailConfig.Content)
		Layout().Render(renderContext, &buffer)
	}

	html := buffer.String()

	message := mailgun.NewMessage(sender, subject, "")
	for _, recipient := range emailConfig.Recipients {
		recipientString := fmt.Sprintf("%s, <%s>", recipient.Name, recipient.Email)
		err = message.AddRecipient(recipientString)
		if err != nil {
			return
		}
	}
	message.SetHTML(html)

	if emailConfig.EmailName != "" {
		message.AddVariable("email", emailConfig.EmailName)
	}

	for _, variable := range emailConfig.Variables {
		message.AddVariable(variable.Key, variable.Value)
	}

	mgCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return mg.Send(mgCtx, message)
}
