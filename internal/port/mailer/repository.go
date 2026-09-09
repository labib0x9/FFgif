package mailer

//go:generate mockgen -source=repository.go -destination=mocks/mock_email_sender.go -package=mocks
type EmailSender interface {
	SendVerificationToken(email string, token string) error
	SendResetPassword(email string, token string) error
	SendResetNotification(email string) error
	SendShareNotification(email string, token string) error
}
