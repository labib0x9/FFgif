package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/model"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type reqSignup struct {
	Username        string `json:"username" validate:"required,min=4,max=20,alphanum"`
	Fullname        string `json:"fullname" validate:"required,min=4,max=100"`
	Email           string `json:"email" validate:"required,email,max=50"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req reqSignup
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Error("Signup: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		// can we be specific what field caused error ?
		http.Error(w, "field required", 422)
		slog.Error("Signup: struct validation failed", "error", err)
		return
	}

	_, err := h.authRepo.GetByEmail(req.Email)
	if err == nil {
		utils.SendJson(w, "email exists", http.StatusConflict)
		slog.Error("Signup: email exists", "error", err, "email", req.Email)
		return
	}

	// what if is internel server error ??

	passHash, err := utils.GenerateHash(req.Password, h.middlewares.Cnf.HashPepper, h.middlewares.Cnf.BcryptCost)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Signup: hash generation failed", "error", err)
		return
	}

	newUser := model.User{
		Username:     req.Username,
		Fullname:     req.Fullname,
		Email:        req.Email,
		PasswordHash: passHash,
		Role:         "user",
		IsVerified:   false,
	}

	createdUser, err := h.authRepo.Create(newUser)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Signup: create user failed", "error", err, "email", newUser.Email)
		return
	}

	verifyToken, verifyTokenHash := utils.GenerateToken()

	newVerifier := model.Verifier{
		UserId: createdUser.Id,
		Token:  verifyTokenHash,
	}

	if err = h.verifierRepo.Create(newVerifier); err != nil {
		// need to think...
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Signup: create verifier failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		if err := h.authRepo.DeleteByEmail(newUser.Email); err != nil {
			slog.Error("Signup: delete user failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		}
		return
	}

	profile := model.Profile{
		UserId:     createdUser.Id,
		ProfilePic: "",
	}

	if err = h.userRepo.SetProfile(profile); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Signup: create profile failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		if err := h.authRepo.DeleteByEmail(newUser.Email); err != nil {
			slog.Error("Signup: delete user failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		}
		return
	}

	quota := model.Quota{
		UserID: createdUser.Id,
	}

	if err := h.quotaRepo.Create(quota); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Signup: quota create failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		return
	}

	// send verification
	if err := h.mailer.SendVerificationToken(newUser.Email, verifyToken); err != nil {
		utils.SendJson(w, "user created, request for resend verification", http.StatusCreated)
		slog.Error("Signup: send verification token failed", "error", err, "email", createdUser.Email, "id", createdUser.Id)
		return
	}

	utils.SendJson(w, "user created", http.StatusCreated)
}
