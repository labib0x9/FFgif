package httputil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	jsonio "github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func TestSendJson(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"status": "ok", "message": "success"}

	jsonio.SendJson(rec, payload, http.StatusOK)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response["status"] != "ok" || response["message"] != "success" {
		t.Errorf("unexpected body payload: %v", response)
	}
}

func TestSendError(t *testing.T) {
	rec := httptest.NewRecorder()
	errorMsg := "Resource not found"

	jsonio.SendError(rec, errorMsg, http.StatusNotFound)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response["error"] != errorMsg {
		t.Errorf("expected error %s, got %v", errorMsg, response["error"])
	}
	if response["code"] != float64(http.StatusNotFound) {
		t.Errorf("expected code %d, got %v", http.StatusNotFound, response["code"])
	}
}
