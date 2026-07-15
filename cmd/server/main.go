package main

import (
	"log"

	"github.com/ChanFrancis/GO_BarterSwap/internal/auth"
	"github.com/ChanFrancis/GO_BarterSwap/internal/config"
	"github.com/ChanFrancis/GO_BarterSwap/internal/database"
	"github.com/ChanFrancis/GO_BarterSwap/internal/handlers"
	"github.com/ChanFrancis/GO_BarterSwap/internal/mailer"
	"github.com/ChanFrancis/GO_BarterSwap/internal/server"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	authService := auth.NewService(db, mailer.SMTP{Addr: cfg.SMTPAddr, From: cfg.EmailFrom}, cfg.AppURL)
	authHandler := &handlers.Auth{Service: authService, SecureCookie: cfg.SecureCookie}

	srv := server.New(cfg, authHandler)

	log.Printf("BarterSwap démarré sur http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
