package server

import (
	"net/http"
	"time"

	"github.com/ChanFrancis/GO_BarterSwap/internal/config"
	"github.com/ChanFrancis/GO_BarterSwap/internal/handlers"
)

// New construit le serveur HTTP avec toutes les routes de l'application.
func New(cfg config.Config, authHandler *handlers.Auth, itemsHandler *handlers.Items, tradesHandler *handlers.Trades) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)

	mux.HandleFunc("POST /api/register", authHandler.Register)
	mux.HandleFunc("POST /api/login", authHandler.Login)
	mux.HandleFunc("POST /api/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/password/forgot", authHandler.ForgotPassword)
	mux.HandleFunc("POST /api/password/reset", authHandler.ResetPassword)

	// Catalogue public
	mux.HandleFunc("GET /api/items", itemsHandler.List)
	mux.HandleFunc("GET /api/items/{id}", itemsHandler.Get)

	// Routes protégées par session
	protected := func(h http.HandlerFunc) http.Handler {
		return authHandler.RequireSession(h)
	}
	mux.Handle("GET /api/me", protected(handlers.Me))
	mux.Handle("POST /api/items", protected(itemsHandler.Create))
	mux.Handle("PUT /api/items/{id}", protected(itemsHandler.Update))
	mux.Handle("DELETE /api/items/{id}", protected(itemsHandler.Delete))
	mux.Handle("POST /api/trades", protected(tradesHandler.Create))
	mux.Handle("GET /api/trades", protected(tradesHandler.List))
	mux.Handle("POST /api/trades/{id}/accept", protected(tradesHandler.Accept))
	mux.Handle("POST /api/trades/{id}/decline", protected(tradesHandler.Decline))
	mux.Handle("POST /api/trades/{id}/cancel", protected(tradesHandler.Cancel))

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           secureHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
