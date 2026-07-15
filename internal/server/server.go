package server

import (
	"net/http"
	"time"

	"github.com/ChanFrancis/GO_BarterSwap/internal/config"
	"github.com/ChanFrancis/GO_BarterSwap/internal/handlers"
)

// New construit le serveur HTTP avec toutes les routes de l'application.
func New(cfg config.Config, authHandler *handlers.Auth) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)

	mux.HandleFunc("POST /api/register", authHandler.Register)
	mux.HandleFunc("POST /api/login", authHandler.Login)
	mux.HandleFunc("POST /api/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/password/forgot", authHandler.ForgotPassword)
	mux.HandleFunc("POST /api/password/reset", authHandler.ResetPassword)

	// Exemple de route protégée ; les routes métier (objets, trocs)
	// utiliseront le même middleware RequireSession.
	mux.Handle("GET /api/me", authHandler.RequireSession(http.HandlerFunc(handlers.Me)))

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           secureHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
