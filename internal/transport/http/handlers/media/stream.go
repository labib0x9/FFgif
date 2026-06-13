package media

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Info("Status: key missing")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	Range := r.Header.Get("Range")

	result, err := h.srv.Stream(r.Context(), key, Range)
	if err != nil {
		switch err {

		}
		return
	}
	defer result.Obj.Close()

	length := result.End - result.Start + 1

	w.Header().Set("Content-Type", result.Info.ContentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set(
		"Content-Range",
		fmt.Sprintf("bytes %d-%d/%d", result.Start, result.End, result.Info.Size),
	)
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`inline; filename="%s"`, key),
	)

	if r.Header.Get("Range") != "" {
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	io.Copy(w, result.Obj)
}
