package handlers

import "net/http"

// Me confirme simplement que la session est valide ; il sera enrichi avec le
// profil de l'utilisateur en Phase 2.
func Me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "session valide"})
}
