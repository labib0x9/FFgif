package mailer

import (
	"github.com/labib0x9/ffgif/config"
	"gopkg.in/gomail.v2"
)

type SmtpMailer struct {
	d    *gomail.Dialer
	from string
}

func NewSmtpMailer(cnf *config.Config) *SmtpMailer {
	return &SmtpMailer{
		d:    gomail.NewDialer(cnf.SMTP.Host, cnf.SMTP.Port, cnf.SMTP.User, cnf.SMTP.Pass),
		from: cnf.SMTP.User,
	}
}

func (m *SmtpMailer) SendVerificationToken(email string, token string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "Verify your account")
	msg.SetBody("text/html", verifyAccountBody(token))

	return m.d.DialAndSend(msg)
}

func (m *SmtpMailer) SendResetPassword(email string, token string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "Reset Password")
	msg.SetBody("text/html", sendPasswordResetBody(token))

	return m.d.DialAndSend(msg)
}

func (m *SmtpMailer) SendResetNotification(email string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "Reset Password")
	msg.SetBody("text/html", `<h1>Alert, your password has been reset</h1>`)

	return m.d.DialAndSend(msg)
}

func (m *SmtpMailer) SendShareNotification(email string, token string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "Share gif")
	msg.SetBody("text/html", sendShareBody(token))

	return m.d.DialAndSend(msg)
}
