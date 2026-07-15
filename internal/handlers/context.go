package handlers

import (
	"context"
	"net/http"
)

type contextKey int

const userIDKey contextKey = 0

// WithUserID attache l'utilisateur authentifié au contexte de la requête.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID retourne l'utilisateur authentifié ; 0 si la requête n'est pas
// passée par RequireSession.
func UserID(r *http.Request) int64 {
	id, _ := r.Context().Value(userIDKey).(int64)
	return id
}
