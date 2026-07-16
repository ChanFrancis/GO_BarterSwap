package api

import (
	"encoding/json"
	"net/http"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

// handleCreateReview laisse un avis sur un échange terminé.
func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	exchangeID, callerID, err := idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in struct {
		Note        int    `json:"note"`
		Commentaire string `json:"commentaire"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := barterswap.ValidateNote(in.Note); err != nil {
		respondError(w, err)
		return
	}
	review, err := s.store.InsertReview(r.Context(), exchangeID, callerID, in.Note, in.Commentaire)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

// handleUserReviews liste les avis reçus par un utilisateur.
func (s *Server) handleUserReviews(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.store.UserExists(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	reviews, err := s.store.ReviewsForUser(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

// handleServiceReviews liste les avis portant sur un service.
func (s *Server) handleServiceReviews(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if _, err := s.store.FetchService(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	reviews, err := s.store.ReviewsForService(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

// handleUserStats retourne le tableau de bord d'un utilisateur.
func (s *Server) handleUserStats(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	stats, err := s.store.FetchUserStats(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
