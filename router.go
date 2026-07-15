package main

import (
	"encoding/json"
	"net/http"
)

// newRouter déclare toutes les routes de l'API. Les handlers HTTP ne
// contiennent aucune logique métier (voir CLAUDE.md).
func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
