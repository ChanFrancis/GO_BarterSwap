package main

import (
	"encoding/json"
	"net/http"
)

// Handlers des évaluations et des statistiques.

// handleCreateReview laisse un avis sur un échange terminé.
func (a *app) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	exchangeID, callerID, err := a.idAndCaller(r)
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
	if err := validerNote(in.Note); err != nil {
		respondError(w, err)
		return
	}
	review, err := a.insertReview(r.Context(), exchangeID, callerID, in.Note, in.Commentaire)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

// handleUserReviews liste les avis reçus par un utilisateur.
func (a *app) handleUserReviews(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := a.userExists(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	reviews, err := a.reviewsForUser(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

// handleServiceReviews liste les avis portant sur un service.
func (a *app) handleServiceReviews(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if _, err := a.fetchService(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	reviews, err := a.reviewsForService(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

// handleUserStats retourne le tableau de bord d'un utilisateur.
func (a *app) handleUserStats(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	stats, err := a.fetchUserStats(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
