package main

import (
	"context"
	"encoding/json"
	"net/http"
)

// Handlers des échanges.

// handleCreateExchange crée une demande d'échange sur un service.
func (a *app) handleCreateExchange(w http.ResponseWriter, r *http.Request) {
	callerID, err := currentUserID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var in struct {
		ServiceID int `json:"service_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if in.ServiceID <= 0 {
		respondError(w, ValidationError{"service_id est obligatoire"})
		return
	}
	ex, err := a.createExchange(r.Context(), callerID, in.ServiceID)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ex)
}

// handleListExchanges liste les échanges de l'utilisateur (demandés + reçus),
// filtrables par ?status=.
func (a *app) handleListExchanges(w http.ResponseWriter, r *http.Request) {
	callerID, err := currentUserID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && !statutValide(status) {
		respondError(w, ValidationError{"statut de filtre inconnu"})
		return
	}
	list, err := a.listExchanges(r.Context(), callerID, status)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetExchange retourne le détail d'un échange (réservé aux parties).
func (a *app) handleGetExchange(w http.ResponseWriter, r *http.Request) {
	id, callerID, err := a.idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	ex, err := a.fetchExchange(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	if callerID != ex.RequesterID && callerID != ex.OwnerID {
		respondError(w, ErrInterdit)
		return
	}
	writeJSON(w, http.StatusOK, ex)
}

func (a *app) handleAcceptExchange(w http.ResponseWriter, r *http.Request) {
	a.exchangeAction(w, r, a.acceptExchange)
}

func (a *app) handleRejectExchange(w http.ResponseWriter, r *http.Request) {
	a.exchangeAction(w, r, a.rejectExchange)
}

func (a *app) handleCompleteExchange(w http.ResponseWriter, r *http.Request) {
	a.exchangeAction(w, r, a.completeExchange)
}

func (a *app) handleCancelExchange(w http.ResponseWriter, r *http.Request) {
	a.exchangeAction(w, r, a.cancelExchange)
}

// exchangeAction factorise les transitions : lecture de l'id et de
// l'appelant, exécution, renvoi de l'échange à jour.
func (a *app) exchangeAction(w http.ResponseWriter, r *http.Request,
	action func(ctx context.Context, id, callerID int) (Exchange, error)) {
	id, callerID, err := a.idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	ex, err := action(r.Context(), id, callerID)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ex)
}
