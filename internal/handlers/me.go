package handlers

import "net/http"

// Me confirme que la session est valide et retourne l'identifiant de
// l'utilisateur connecté.
func Me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"user_id": UserID(r)})
}
