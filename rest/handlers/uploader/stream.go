package uploader

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Info("Status: key missing")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	info, err := h.uploaderRepo.StatObject(r.Context(), key)
	if err != nil {
		slog.Info("Stream: info missed")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	start, end, err := parseRange(r.Header.Get("Range"), info.Size)
	if err != nil {
		slog.Info("Stream", "err", err)
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	obj, err := h.uploaderRepo.GetObject(r.Context(), start, end, key)
	if err != nil {
		http.Error(w, "stream failed", http.StatusInternalServerError)
		return
	}
	defer obj.Close()

	length := end - start + 1

	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set(
		"Content-Range",
		fmt.Sprintf("bytes %d-%d/%d", start, end, info.Size),
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

	io.Copy(w, obj)
}

func parseRange(r string, total int64) (start int64, end int64, err error) {

	if r == "" {
		return 0, total - 1, nil
	}

	if !strings.HasPrefix(r, "bytes=") {
		return 0, 0, errors.New("invalid range")
	}

	parts := strings.Split(strings.TrimPrefix(r, "bytes="), "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid range")
	}

	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	if parts[1] == "" {
		end = total - 1
	} else {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}

	if end >= total {
		end = total - 1
	}

	if start > end {
		return 0, 0, errors.New("range out of bounds")
	}

	return
}
