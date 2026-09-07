package mailer

import "fmt"

func verifyAccountBody(token string) string {
	url := fmt.Sprintf("http://127.0.0.1:8080/auth/verify?token=%s", token)
	return fmt.Sprintf(`
			<h1>Welcome To FFgif</h1>
            <p>Click the link below to verify your account.</p>
			<button>
            <a href="%s">Verify my account</a>
			</button>
            <p>This link expires in 30 minutes.</p>
        `, url)
}

func sendPasswordResetBody(token string) string {
	url := fmt.Sprintf("http://127.0.0.1:8080/auth/reset?token=%s", token)
	return fmt.Sprintf(`
			<h1>FFgif Password Reset</h1>
            <p>Click the link below to reset your password.</p>
			<button>
            <a href="%s">reset password</a>
			</button>
            <p>This link expires in 15 minutes.</p>
        `, url)
}

func sendShareBody(token string) string {
	url := fmt.Sprintf("http://127.0.0.1:8080/s/%s", token)
	return fmt.Sprintf(`
			<h1>FFgif Gif Share</h1>
            <p>Click the link below to download the shared GIF.</p>
			<button>
            <a href="%s">GIF</a>
			</button>
        `, url)
}
