package share

func (h *service) Create() {
	// id := middleware.GetUserId(r)
	// if id == "" {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("CreateShare: id not found")
	// 	return
	// }

	// gifId := r.PathValue("id")

	// var body struct {
	// 	ExpiresIn int    `json:"expires_in"` // hours, 0 = never
	// 	Access    string `json:"access"`     // "view" or "download"
	// }
	// if err := utils.ParseJson(r, &body); err != nil {
	// 	http.Error(w, "bad request", http.StatusBadRequest)
	// 	slog.Error("CreateShare: ParseJson() failed", "error", err)
	// 	return
	// }
	// if body.Access == "" {
	// 	body.Access = "view"
	// }

	// share, err := h.shareRepo.Create(id, gifId, body.Access, body.ExpiresIn)
	// if err != nil {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("CreateShare: CreateShare() failed", "error", err)
	// 	return
	// }

	// utils.SendJson(w, share, http.StatusCreated)
}
