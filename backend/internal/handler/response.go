package handler

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse คือรูปแบบ JSON มาตรฐานเวลา request ล้มเหลว ใช้เป็น response schema ใน swagger docs
type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
