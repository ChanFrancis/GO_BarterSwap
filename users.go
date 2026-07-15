package main

import (
	"encoding/json"
	"net/http"
)

// Handlers users et skills : décodage/encodage JSON et contrôle d'accès
// uniquement, les règles métier sont dans business.go.

type userInput struct {
	Pseudo string `json:"pseudo"`
	Bio    string `json:"bio"`
	Ville  string `json:"ville"`
}

// handleCreateUser crée un compte ; les crédits de bienvenue sont attribués
// automatiquement.
func (a *app) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in userInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := validerPseudo(in.Pseudo); err != nil {
		respondError(w, err)
		return
	}
	user, err := a.insertUser(r.Context(), in.Pseudo, in.Bio, in.Ville)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// handleGetUser retourne le profil public d'un utilisateur.
func (a *app) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	user, err := a.fetchUser(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// handleUpdateUser modifie son propre profil (X-User-ID doit correspondre).
func (a *app) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := a.selfOnly(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in userInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := validerPseudo(in.Pseudo); err != nil {
		respondError(w, err)
		return
	}
	if err := a.updateUser(r.Context(), id, in.Pseudo, in.Bio, in.Ville); err != nil {
		respondError(w, err)
		return
	}
	user, err := a.fetchUser(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// handleGetSkills liste les compétences d'un utilisateur.
func (a *app) handleGetSkills(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := a.userExists(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	skills, err := a.fetchSkills(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

// handlePutSkills définit ses compétences (écrase la liste existante).
func (a *app) handlePutSkills(w http.ResponseWriter, r *http.Request) {
	id, err := a.selfOnly(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var skills []Skill
	if err := json.NewDecoder(r.Body).Decode(&skills); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := validerSkills(skills); err != nil {
		respondError(w, err)
		return
	}
	if err := a.replaceSkills(r.Context(), id, skills); err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

// selfOnly vérifie que l'appelant (X-User-ID) agit sur sa propre ressource
// {id} et que celle-ci existe.
func (a *app) selfOnly(r *http.Request) (int, error) {
	id, err := pathID(r)
	if err != nil {
		return 0, err
	}
	caller, err := currentUserID(r)
	if err != nil {
		return 0, err
	}
	if caller != id {
		return 0, ErrInterdit
	}
	if err := a.userExists(r.Context(), id); err != nil {
		return 0, err
	}
	return id, nil
}
