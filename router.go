package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// routes déclare toutes les routes de l'API. Les handlers HTTP ne
// contiennent aucune logique métier (voir business.go).
func (a *app) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	mux.HandleFunc("POST /api/users", a.handleCreateUser)
	mux.HandleFunc("GET /api/users/{id}", a.handleGetUser)
	mux.HandleFunc("PUT /api/users/{id}", a.handleUpdateUser)
	mux.HandleFunc("GET /api/users/{id}/skills", a.handleGetSkills)
	mux.HandleFunc("PUT /api/users/{id}/skills", a.handlePutSkills)

	mux.HandleFunc("GET /api/services", a.handleListServices)
	mux.HandleFunc("POST /api/services", a.handleCreateService)
	mux.HandleFunc("GET /api/services/{id}", a.handleGetService)
	mux.HandleFunc("PUT /api/services/{id}", a.handleUpdateService)
	mux.HandleFunc("DELETE /api/services/{id}", a.handleDeleteService)

	return withMiddlewares(mux)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// respondError traduit les erreurs métier en codes HTTP, sans jamais
// exposer une erreur interne au client.
func respondError(w http.ResponseWriter, err error) {
	var validation ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusBadRequest, validation.Message)
	case errors.Is(err, ErrIntrouvable):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInterdit):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrCompetenceManquante):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("erreur interne : %v", err)
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}
