package mailer

type EmailSender interface {
	SendVerificationToken(email string, token string) error
	SendResetPassword(email string, token string) error
	SendResetNotification(email string) error
}
