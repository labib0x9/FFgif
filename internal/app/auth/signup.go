package auth

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/token"
)

type SignupResult struct {
}

func (s *service) Signup(ctx context.Context, email string, username string, fullname string, password string) (*SignupResult, error) {
	_, err := s.authRepo.GetByEmail(email)
	if err == nil {
		// utils.SendJson(w, "email exists", http.StatusConflict)
		// slog.Error("Signup: email exists", "error", err, "email", req.Email)
		return nil, auth.ErrUserExits
	}

	// what if is internel server error ??

	passHash, err := s.hasher.GenerateHash(password)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Signup: hash generation failed", "error", err)
		return nil, auth.ErrHashGenFailed
	}

	newUser := auth.User{
		Username:     username,
		Fullname:     fullname,
		Email:        email,
		PasswordHash: passHash,
		Role:         "user",
		IsVerified:   false,
	}

	createdUser, err := s.authRepo.Create(newUser)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Signup: create user failed", "error", err, "email", newUser.Email)
		return nil, auth.ErrUserCreateFailed
	}

	verifyToken, verifyTokenHash := token.GenerateToken()

	newVerifier := auth.Verifier{
		UserId: createdUser.Id,
		Token:  verifyTokenHash,
	}

	if err = s.verifierRepo.Create(newVerifier); err != nil {
		// need to think...
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Signup: create verifier failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		// if err := h.authRepo.DeleteByEmail(newUser.Email); err != nil {
		// 	slog.Error("Signup: delete user failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		// }
		return nil, auth.ErrVerifierTokenCreateFailed
	}

	profile := user.Profile{
		UserId:     createdUser.Id,
		ProfilePic: "",
	}

	if err = s.profileRepo.SetProfile(profile); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Signup: create profile failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		// if err := h.authRepo.DeleteByEmail(newUser.Email); err != nil {
		// 	slog.Error("Signup: delete user failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		// }
		return nil, auth.ErrSetProfileFailed
	}

	quota := user.Quota{
		UserID: createdUser.Id,
	}

	if err := s.quotaRepo.Create(quota); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Signup: quota create failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		return nil, auth.ErrQuotaCreateFailed
	}

	// send verification
	// if err := h.mailer.SendVerificationToken(newUser.Email, verifyToken); err != nil {
	// 	utils.SendJson(w, "user created, request for resend verification", http.StatusCreated)
	// 	slog.Error("Signup: send verification token failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
	// 	return
	// }

	mqMsg := queue.EmailMessage{
		To:    newUser.Email,
		Name:  "signup",
		Token: verifyToken,
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		// utils.SendJson(w, "user created, request for resend verification", http.StatusCreated)
		// slog.Error("Signup: send verification token failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		return nil, auth.ErrMessageQueueFailed
	}
	return &SignupResult{}, nil
}
