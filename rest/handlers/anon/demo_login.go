package anon

// import (
// 	"log/slog"
// 	"net/http"

// 	"github.com/labib0x9/ProjectUnsafe/model"
// 	"github.com/labib0x9/ProjectUnsafe/utils"
// )

// func (h *Handler) DemoLogin(w http.ResponseWriter, r *http.Request) {
// 	id := utils.GenerateRandomID()
// 	newUser := model.AnonUser{
// 		Id:       id,
// 		Username: id.String()[:15],
// 		Fullname: id.String(),
// 	}

// 	createdUser, err := h.authRepo.CreateDemo(newUser)
// 	if err != nil {
// 		http.Error(w, "internal server error", http.StatusInternalServerError)
// 		slog.Error("Signup: create user failed", "error", err, "id", id.String())
// 		return
// 	}

// 	profile := model.Profile{
// 		UserId:     createdUser.Id,
// 		ProfilePic: "",
// 	}

// 	if err = h.userRepo.SetProfileDemo(profile); err != nil {
// 		http.Error(w, "internal server error", http.StatusInternalServerError)
// 		slog.Error("Signup: create profile failed", "error", err, "id", createdUser.Id)
// 		return
// 	}

// 	quota := model.Quota{
// 		UserID: createdUser.Id,
// 	}

// 	if err := h.quotaRepo.CreateDemo(quota); err != nil {
// 		http.Error(w, "internal server error", http.StatusInternalServerError)
// 		slog.Error("Signup: quota create failed", "error", err, "id", createdUser.Id)
// 		return
// 	}

// 	token, err := utils.CreateJWT(h.middlewares.Cnf.JwtSecret, createdUser)
// 	if err != nil {
// 		http.Error(w, "internal server error", http.StatusInternalServerError)
// 		slog.Error("Login: jwt create error", "error", err)
// 		return
// 	}

// 	utils.SendJson(w, map[string]any{
// 		"token":      token,
// 		"id":         createdUser.Id,
// 		"expires_at": createdUser.DeletedAt,
// 	}, http.StatusOK)
// }
