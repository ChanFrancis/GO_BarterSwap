package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

// handleCreateExchange crée une demande d'échange sur un service.
func (s *Server) handleCreateExchange(w http.ResponseWriter, r *http.Request) {
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
		respondError(w, barterswap.ValidationError{Message: "service_id est obligatoire"})
		return
	}
	ex, err := s.store.CreateExchange(r.Context(), callerID, in.ServiceID)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ex)
}

// handleListExchanges liste les échanges de l'utilisateur (demandés + reçus),
// filtrables par ?status=.
func (s *Server) handleListExchanges(w http.ResponseWriter, r *http.Request) {
	callerID, err := currentUserID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && !barterswap.ValidStatus(status) {
		respondError(w, barterswap.ValidationError{Message: "statut de filtre inconnu"})
		return
	}
	list, err := s.store.ListExchanges(r.Context(), callerID, status)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetExchange retourne le détail d'un échange (réservé aux parties).
func (s *Server) handleGetExchange(w http.ResponseWriter, r *http.Request) {
	id, callerID, err := idAndCaller(r)
	if err != nil {
		respondError(w, err)
		return
	}
	ex, err := s.store.FetchExchange(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	if callerID != ex.RequesterID && callerID != ex.OwnerID {
		respondError(w, barterswap.ErrInterdit)
		return
	}
	writeJSON(w, http.StatusOK, ex)
}

func (s *Server) handleAcceptExchange(w http.ResponseWriter, r *http.Request) {
	s.exchangeAction(w, r, s.store.AcceptExchange)
}

func (s *Server) handleRejectExchange(w http.ResponseWriter, r *http.Request) {
	s.exchangeAction(w, r, s.store.RejectExchange)
}

func (s *Server) handleCompleteExchange(w http.ResponseWriter, r *http.Request) {
	s.exchangeAction(w, r, s.store.CompleteExchange)
}

func (s *Server) handleCancelExchange(w http.ResponseWriter, r *http.Request) {
	s.exchangeAction(w, r, s.store.CancelExchange)
}

// exchangeAction factorise les transitions : lecture de l'id et de
// l'appelant, exécution, renvoi de l'échange à jour.
func (s *Server) exchangeAction(w http.ResponseWriter, r *http.Request,
	action func(ctx context.Context, id, callerID int) (barterswap.Exchange, error)) {
	id, callerID, err := idAndCaller(r)
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
