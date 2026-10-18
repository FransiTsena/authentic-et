package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/authentic-et/authentic-et/internal/domain"
)

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		WriteJSON(w, appErr.StatusCode, map[string]any{
			"code":    appErr.Code,
			"message": appErr.Message,
		})
		return
	}

	if errors.Is(err, domain.ErrNotFound) {
		WriteJSON(w, http.StatusNotFound, map[string]string{
			"code":    "NOT_FOUND",
			"message": "Resource not found",
		})
		return
	}

	if errors.Is(err, domain.ErrTransactionAlreadyClaimed) {
		WriteJSON(w, http.StatusConflict, map[string]string{
			"code":    "TRANSACTION_ALREADY_CLAIMED",
			"message": "Transaction has already been claimed",
		})
		return
	}

	if errors.Is(err, domain.ErrUnauthorized) {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"code":    "UNAUTHORIZED",
			"message": "Unauthorized access",
		})
		return
	}

	WriteJSON(w, http.StatusInternalServerError, map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An unexpected error occurred. Please contact support or check server logs.",
	})
}

func ReadJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	// Enforce 1 MB maximum payload limit to prevent unbounded memory DoS
	limitedReader := io.LimitReader(r.Body, 1<<20)
	return json.NewDecoder(limitedReader).Decode(target)
}

func GetQueryInt(r *http.Request, key string, defaultVal int) int {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return v
}
