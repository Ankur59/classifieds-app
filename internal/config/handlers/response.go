package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// writeJSON encodes v as the JSON response body with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("handlers.writeJSON: %v", err)
	}
}

// writeError sends {"error": msg} with the given status.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
