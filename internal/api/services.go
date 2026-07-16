package api

import (
	"encoding/json"
	"net/http"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
	"github.com/ChanFrancis/GO_BarterSwap/internal/store"
)

// handleCreateService publie une annonce. L'utilisateur doit posséder une
// compétence correspondant à la catégorie.
func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	callerID, err := currentUserID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in barterswap.ServiceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := barterswap.ValidateService(in); err != nil {
		respondError(w, err)
		return
	}
	if err := s.store.UserExists(r.Context(), callerID); err != nil {
		respondError(w, err)
		return
	}
	hasSkill, err := s.store.UserHasSkill(r.Context(), callerID, in.Categorie)
	if err != nil {
		respondError(w, err)
		return
	}
	if !hasSkill {
		respondError(w, barterswap.ErrCompetenceManquante)
		return
	}
	service, err := s.store.InsertService(r.Context(), callerID, in)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, service)
}

// handleListServices liste les annonces actives, filtrées côté serveur.
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	services, err := s.store.ListServices(r.Context(), store.ServiceFilter{
		Categorie: q.Get("categorie"),
		Ville:     q.Get("ville"),
		Search:    q.Get("search"),
	})
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, services)
}

// handleGetService retourne le détail d'une annonce.
func (s *Server) handleGetService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	service, err := s.store.FetchService(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, service)
}

// handleUpdateService modifie sa propre annonce.
func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id, callerID, err := idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in barterswap.ServiceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := barterswap.ValidateService(in); err != nil {
		respondError(w, err)
		return
	}
	service, err := s.store.UpdateService(r.Context(), id, callerID, in)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, service)
}

// handleDeleteService supprime sa propre annonce.
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id, callerID, err := idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.store.DeleteService(r.Context(), id, callerID); err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "annonce supprimée"})
}
