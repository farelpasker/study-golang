package utils

import (
	"encoding/json"
	"net/http"
)

func Success(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func Error(w http.ResponseWriter, statusCode int, message string) {
	Success(w, statusCode, map[string]string{"error": message})
}