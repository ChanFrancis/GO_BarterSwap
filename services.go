package main

import (
	"encoding/json"
	"net/http"
)

// Handlers des annonces de services.

// handleCreateService publie une annonce. L'utilisateur doit posséder une
// compétence correspondant à la catégorie.
func (a *app) handleCreateService(w http.ResponseWriter, r *http.Request) {
	callerID, err := currentUserID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in serviceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := validerService(in); err != nil {
		respondError(w, err)
		return
	}
	if err := a.userExists(r.Context(), callerID); err != nil {
		respondError(w, err)
		return
	}
	hasSkill, err := a.userHasSkill(r.Context(), callerID, in.Categorie)
	if err != nil {
		respondError(w, err)
		return
	}
	if !hasSkill {
		respondError(w, ErrCompetenceManquante)
		return
	}
	service, err := a.insertService(r.Context(), callerID, in)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, service)
}

// handleListServices liste les annonces actives, filtrées côté serveur.
func (a *app) handleListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	services, err := a.listServices(r.Context(), serviceFilter{
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
func (a *app) handleGetService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	service, err := a.fetchService(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, service)
}

// handleUpdateService modifie sa propre annonce.
func (a *app) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id, callerID, err := a.idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in serviceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := validerService(in); err != nil {
		respondError(w, err)
		return
	}
	service, err := a.updateService(r.Context(), id, callerID, in)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, service)
}

// handleDeleteService supprime sa propre annonce.
func (a *app) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id, callerID, err := a.idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := a.deleteService(r.Context(), id, callerID); err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "annonce supprimée"})
}

// idAndCaller lit l'identifiant {id} du chemin et l'appelant (X-User-ID).
func (a *app) idAndCaller(r *http.Request) (id, callerID int, err error) {
	if id, err = pathID(r); err != nil {
		return 0, 0, err
	}
	if callerID, err = currentUserID(r); err != nil {
		return 0, 0, err
	}
	return id, callerID, nil
}
