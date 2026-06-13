package mailer

import (
	"fmt"
	"net/smtp"

	"github.com/labib0x9/ProjectUnsafe/config"
)

type Mailer struct {
	email        string
	mailtrapUser string
	mailtrapPass string
}

func NewMailer(cnf *config.Config) *Mailer {
	return &Mailer{
		email:        cnf.Email,
		mailtrapUser: cnf.MailtrapUser,
		mailtrapPass: cnf.MailtrapPass,
	}
}

func (m *Mailer) SendVerificationToken(email string, token string) error {
	from := m.email
	username := m.mailtrapUser
	password := m.mailtrapPass

	smtpHost := "sandbox.smtp.mailtrap.io"
	smtpPort := "587"

	subject := "Verify your email"

	url := fmt.Sprintf("http://127.0.0.1:8080/auth/verify?token=%s", token)
	body :=
		fmt.Sprintf(`
			<h1>Welcome To ProjectPDF</h1>
            <p>Click the link below to verify your account.</p>
			<button>
            <a href="%s">Verify my account</a>
			</button>
            <p>This link expires in 30 minutes.</p>
        `, url)

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

func (m *Mailer) SendResetPassword(email string, token string) error {
	from := m.email
	username := m.mailtrapUser
	password := m.mailtrapPass

	smtpHost := "sandbox.smtp.mailtrap.io"
	smtpPort := "587"

	subject := "Reset Password"

	url := fmt.Sprintf("http://127.0.0.1:8080/auth/reset?token=%s", token)
	body :=
		fmt.Sprintf(`
			<h1>ProjectPDF PAssword Reset</h1>
            <p>Click the link below to reset your password.</p>
			<button>
            <a href="%s">reset password</a>
			</button>
            <p>This link expires in 15 minutes.</p>
        `, url)

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

func (m *Mailer) SendResetNotification(email string) error {
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
