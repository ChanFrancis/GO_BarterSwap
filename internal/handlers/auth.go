package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ChanFrancis/GO_BarterSwap/internal/auth"
)

const sessionCookie = "barterswap_session"

// Auth expose les endpoints d'authentification en JSON.
type Auth struct {
	Service      *auth.Service
	SecureCookie bool // true en production (HTTPS)
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *Auth) Register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := a.Service.Register(r.Context(), c.Email, c.Password); err != nil {
		a.writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "compte créé"})
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	token, err := a.Service.Login(r.Context(), c.Email, c.Password)
	if err != nil {
		a.writeAuthError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: true,
		Secure:   a.SecureCookie,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "connecté"})
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if err := a.Service.Logout(r.Context(), cookie.Value); err != nil {
			log.Printf("déconnexion : %v", err)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.SecureCookie,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "déconnecté"})
}

func (a *Auth) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := a.Service.RequestPasswordReset(r.Context(), body.Email); err != nil {
		// L'erreur est journalisée mais la réponse reste identique pour ne
		// pas révéler si l'email est inscrit.
		log.Printf("demande de réinitialisation : %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "si un compte existe avec cet email, un lien de réinitialisation a été envoyé",
	})
}

func (a *Auth) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := a.Service.ResetPassword(r.Context(), body.Token, body.NewPassword); err != nil {
		a.writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "mot de passe réinitialisé"})
}

// RequireSession est un middleware qui n'autorise que les requêtes portant
// une session valide.
func (a *Auth) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentification requise")
			return
		}
		userID, err := a.Service.UserIDFromSession(r.Context(), cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "session invalide ou expirée")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
	})
}

// writeAuthError traduit les erreurs du service en codes HTTP sans jamais
// exposer d'erreur interne au client.
func (a *Auth) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrAccountLocked):
		writeError(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, auth.ErrPasswordExpired):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrEmailTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrWeakPassword), errors.Is(err, auth.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrInvalidToken):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("erreur interne : %v", err)
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
