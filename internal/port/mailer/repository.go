//go:generate mockgen -package=mocks -destination=mocks/mock_mailer.go github.com/labib0x9/ffgif/internal/port/mailer EmailSender

package mailer

type EmailSender interface {
	SendVerificationToken(email string, token string) error
	SendResetPassword(email string, token string) error
	SendResetNotification(email string) error
	SendShareNotification(email string, token string) error
}
