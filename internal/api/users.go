package api

import (
	"encoding/json"
	"net/http"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

type userInput struct {
	Pseudo string `json:"pseudo"`
	Bio    string `json:"bio"`
	Ville  string `json:"ville"`
}

// handleCreateUser crée un compte ; les crédits de bienvenue sont attribués
// automatiquement.
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in userInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := barterswap.ValidatePseudo(in.Pseudo); err != nil {
		respondError(w, err)
		return
	}
	user, err := s.store.InsertUser(r.Context(), in.Pseudo, in.Bio, in.Ville)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// handleGetUser retourne le profil public d'un utilisateur.
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	user, err := s.store.FetchUser(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// handleUpdateUser modifie son propre profil (X-User-ID doit correspondre).
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := s.selfOnly(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in userInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := barterswap.ValidatePseudo(in.Pseudo); err != nil {
		respondError(w, err)
		return
	}
	if err := s.store.UpdateUser(r.Context(), id, in.Pseudo, in.Bio, in.Ville); err != nil {
		respondError(w, err)
		return
	}
	user, err := s.store.FetchUser(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// handleGetSkills liste les compétences d'un utilisateur.
func (s *Server) handleGetSkills(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.store.UserExists(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	skills, err := s.store.FetchSkills(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

// handlePutSkills définit ses compétences (écrase la liste existante).
func (s *Server) handlePutSkills(w http.ResponseWriter, r *http.Request) {
	id, err := s.selfOnly(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var skills []barterswap.Skill
	if err := json.NewDecoder(r.Body).Decode(&skills); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := barterswap.ValidateSkills(skills); err != nil {
		respondError(w, err)
		return
	}
	if err := s.store.ReplaceSkills(r.Context(), id, skills); err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

// selfOnly vérifie que l'appelant (X-User-ID) agit sur sa propre ressource
// {id} et que celle-ci existe.
func (s *Server) selfOnly(r *http.Request) (int, error) {
	id, err := pathID(r)
	if err != nil {
		return 0, err
	}
	caller, err := currentUserID(r)
	if err != nil {
		return 0, err
	}
	if caller != id {
		return 0, barterswap.ErrInterdit
	}
	if err := s.store.UserExists(r.Context(), id); err != nil {
		return 0, err
	}
	return id, nil
}
