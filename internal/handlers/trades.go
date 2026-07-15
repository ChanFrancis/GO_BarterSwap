package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ChanFrancis/GO_BarterSwap/internal/items"
	"github.com/ChanFrancis/GO_BarterSwap/internal/trades"
)

// Trades expose les offres de troc. Toutes les routes exigent une session.
type Trades struct {
	Service *trades.Service
}

func (h *Trades) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RequestedItemID int64   `json:"requested_item_id"`
		OfferedItemIDs  []int64 `json:"offered_item_ids"`
		Message         string  `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	offer, err := h.Service.Create(r.Context(), UserID(r),
		body.RequestedItemID, body.OfferedItemIDs, body.Message)
	if err != nil {
		writeTradeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, offer)
}

func (h *Trades) List(w http.ResponseWriter, r *http.Request) {
	sent, received, err := h.Service.ListForUser(r.Context(), UserID(r))
	if err != nil {
		writeTradeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sent": sent, "received": received})
}

func (h *Trades) Accept(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, h.Service.Accept, "offre acceptée, objets échangés")
}

func (h *Trades) Decline(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, h.Service.Decline, "offre refusée")
}

func (h *Trades) Cancel(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, h.Service.Cancel, "offre annulée")
}

func (h *Trades) decide(w http.ResponseWriter, r *http.Request,
	action func(ctx context.Context, userID, offerID int64) error, message string) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	if err := action(r.Context(), UserID(r), id); err != nil {
		writeTradeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": message})
}

func writeTradeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, trades.ErrNotFound), errors.Is(err, items.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, trades.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, trades.ErrNotPending),
		errors.Is(err, trades.ErrUnavailable):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, trades.ErrOwnItem),
		errors.Is(err, trades.ErrItemsRequired),
		errors.Is(err, trades.ErrNotYourItems):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("erreur interne : %v", err)
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}
