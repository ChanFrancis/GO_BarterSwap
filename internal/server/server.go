package server

import (
	"net/http"
	"time"

	"github.com/ChanFrancis/GO_BarterSwap/internal/config"
	"github.com/ChanFrancis/GO_BarterSwap/internal/handlers"
)

// New construit le serveur HTTP avec toutes les routes de l'application.
func New(cfg config.Config) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
