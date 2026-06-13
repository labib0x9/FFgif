package jsonio

import (
	"encoding/json"
	"net/http"
)

func SendJson(w http.ResponseWriter, v any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		http.Error(w, "Internal Server Error-Encode.", http.StatusInternalServerError)
		return
	}
}
