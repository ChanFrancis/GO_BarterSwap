// Package api expose l'application en HTTP : routage, middlewares, handlers.
// Les handlers ne portent aucune règle métier ; ils décodent la requête,
// appellent le store et encodent la réponse.
package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
	"github.com/ChanFrancis/GO_BarterSwap/internal/store"
)

// Server porte les dépendances des handlers.
type Server struct {
	store *store.Store
}

// NewServer construit le serveur à partir d'un store.
func NewServer(st *store.Store) *Server {
	return &Server{store: st}
}

// Routes déclare toutes les routes et enveloppe le tout dans les middlewares.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	// Documentation interactive (Swagger UI) et spécification OpenAPI.
	mux.Handle("GET /docs/", swaggerHandler())
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /openapi.yaml", s.handleOpenAPISpec)

	mux.HandleFunc("POST /api/users", s.handleCreateUser)
	mux.HandleFunc("GET /api/users/{id}", s.handleGetUser)
	mux.HandleFunc("PUT /api/users/{id}", s.handleUpdateUser)
	mux.HandleFunc("GET /api/users/{id}/skills", s.handleGetSkills)
	mux.HandleFunc("PUT /api/users/{id}/skills", s.handlePutSkills)

	mux.HandleFunc("GET /api/services", s.handleListServices)
	mux.HandleFunc("POST /api/services", s.handleCreateService)
	mux.HandleFunc("GET /api/services/{id}", s.handleGetService)
	mux.HandleFunc("PUT /api/services/{id}", s.handleUpdateService)
	mux.HandleFunc("DELETE /api/services/{id}", s.handleDeleteService)

	mux.HandleFunc("POST /api/exchanges", s.handleCreateExchange)
	mux.HandleFunc("GET /api/exchanges", s.handleListExchanges)
	mux.HandleFunc("GET /api/exchanges/{id}", s.handleGetExchange)
	mux.HandleFunc("PUT /api/exchanges/{id}/accept", s.handleAcceptExchange)
	mux.HandleFunc("PUT /api/exchanges/{id}/reject", s.handleRejectExchange)
	mux.HandleFunc("PUT /api/exchanges/{id}/complete", s.handleCompleteExchange)
	mux.HandleFunc("PUT /api/exchanges/{id}/cancel", s.handleCancelExchange)

	mux.HandleFunc("POST /api/exchanges/{id}/review", s.handleCreateReview)
	mux.HandleFunc("GET /api/users/{id}/reviews", s.handleUserReviews)
	mux.HandleFunc("GET /api/services/{id}/reviews", s.handleServiceReviews)
	mux.HandleFunc("GET /api/users/{id}/stats", s.handleUserStats)

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

// respondError traduit les erreurs du domaine en codes HTTP, sans jamais
// exposer une erreur interne au client.
func respondError(w http.ResponseWriter, err error) {
	var validation barterswap.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusBadRequest, validation.Message)
	case errors.Is(err, barterswap.ErrCompetenceManquante),
		errors.Is(err, barterswap.ErrServicePropre),
		errors.Is(err, barterswap.ErrCreditsInsuffisants),
		errors.Is(err, barterswap.ErrEchangeNonTermine),
		errors.Is(err, barterswap.ErrDejaNote):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, barterswap.ErrDejaReserve),
		errors.Is(err, barterswap.ErrTransitionInvalide):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, barterswap.ErrIntrouvable):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, barterswap.ErrInterdit):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		log.Printf("erreur interne : %v", err)
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}
