package friend

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/friend"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type reqSendRequest struct {
	AddresseeID string `json:"addressee_id" validate:"required,uuid4"`
}

func (h *Handler) SendRequest(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unauthenticated", http.StatusUnauthorized)
		slog.Error("friend handler - SendRequest() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	var req reqSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		slog.Warn("friend handler - SendRequest() = bad json body", "request_id", reqId, "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("friend handler - SendRequest() = struct validation failed", "request_id", reqId, "error", err)
		return
	}

	f, err := h.srv.SendRequest(r.Context(), userId, req.AddresseeID)
	if err != nil {
		switch {
		case errors.Is(err, friend.ErrCannotFriendSelf):
			httputil.SendError(w, httputil.VALIDATION_FAILED, "cannot send a friend request to yourself", http.StatusUnprocessableEntity)
		case errors.Is(err, friend.ErrAlreadyExists):
			httputil.SendError(w, "FRIENDSHIP_ALREADY_EXISTS", "friendship already exists", http.StatusConflict)
		case errors.Is(err, friend.ErrNotFound):
			httputil.SendError(w, "FRIENDSHIP_NOT_FOUND", "not found", http.StatusNotFound)
		case errors.Is(err, friend.ErrNotAuthorized):
			httputil.SendError(w, "FRIENDSHIP_FORBIDDEN", "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("friend handler - SendRequest()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, f, http.StatusCreated)
}
