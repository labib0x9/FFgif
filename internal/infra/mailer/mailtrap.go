package mailer

import (
	"net/smtp"

	"github.com/labib0x9/ffgif/config"
)

type Mailtrap struct {
	email        string
	mailtrapUser string
	mailtrapPass string
}

func NewMailtrap(cnf *config.Config) *Mailtrap {
	return &Mailtrap{
		email:        cnf.Email,
		mailtrapUser: cnf.Mailtrap.User,
		mailtrapPass: cnf.Mailtrap.Pass,
	}
}

func (m *Mailtrap) SendVerificationToken(email string, token string) error {
	from := m.email
	username := m.mailtrapUser
	password := m.mailtrapPass
	smtpHost := "sandbox.smtp.mailtrap.io"
	smtpPort := "587"
	subject := "Verify your email"
	body := verifyAccountBody(token)
	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + email + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	auth := smtp.PlainAuth("", username, password, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{email}, msg)
}

func (m *Mailtrap) SendResetPassword(email string, token string) error {
	from := m.email
	username := m.mailtrapUser
	password := m.mailtrapPass
	smtpHost := "sandbox.smtp.mailtrap.io"
	smtpPort := "587"
	subject := "Reset Password"
	body := sendPasswordResetBody(token)
	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + email + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	auth := smtp.PlainAuth("", username, password, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{email}, msg)
}

func (m *Mailtrap) SendResetNotification(email string) error {
	from := m.email
	username := m.mailtrapUser
	password := m.mailtrapPass

	smtpHost := "sandbox.smtp.mailtrap.io"
	smtpPort := "587"

	subject := "Reset Password"

	body := `<h1>Alert, your password has been reset</h1>`

	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + email + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	auth := smtp.PlainAuth("", username, password, smtpHost)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{email}, msg)
}
