package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/ChanFrancis/GO_BarterSwap/internal/items"
)

// Items expose le CRUD et la recherche des objets à troquer.
type Items struct {
	Service *items.Service
}

func (h *Items) Create(w http.ResponseWriter, r *http.Request) {
	var in items.Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	item, err := h.Service.Create(r.Context(), UserID(r), in)
	if err != nil {
		writeItemError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Items) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	item, err := h.Service.Get(r.Context(), id)
	if err != nil {
		writeItemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Items) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	var in items.Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	item, err := h.Service.Update(r.Context(), UserID(r), id, in)
	if err != nil {
		writeItemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Items) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	if err := h.Service.Delete(r.Context(), UserID(r), id); err != nil {
		writeItemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "objet supprimé"})
}

// List est publique : c'est le catalogue des objets disponibles.
// Filtres : ?category=, ?q=, ?owner_id=, ?page=
func (h *Items) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	ownerID, _ := strconv.ParseInt(query.Get("owner_id"), 10, 64)

	list, err := h.Service.List(r.Context(), items.Filter{
		Category: query.Get("category"),
		Query:    query.Get("q"),
		OwnerID:  ownerID,
		Page:     page,
	})
	if err != nil {
		writeItemError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list, "page": max(page, 1)})
}

func writeItemError(w http.ResponseWriter, err error) {
	var validation items.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusBadRequest, validation.Error())
	case errors.Is(err, items.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, items.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		log.Printf("erreur interne : %v", err)
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
