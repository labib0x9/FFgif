package httputil

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func SendJson(w http.ResponseWriter, v any, statusCode int) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(v); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	buf.WriteTo(w)
}

func SendError(w http.ResponseWriter, errorCode string, message string, statusCode int) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(map[string]any{
		"error_code": errorCode,
		"message":    message,
		"status":     statusCode,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	buf.WriteTo(w)
}
