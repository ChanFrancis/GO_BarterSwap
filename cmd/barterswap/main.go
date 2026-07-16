// Commande barterswap : point d'entrée de l'API d'échange de compétences.
// Elle câble le store (PostgreSQL) et le serveur HTTP, puis écoute.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ChanFrancis/GO_BarterSwap/internal/api"
	"github.com/ChanFrancis/GO_BarterSwap/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://barterswap:barterswap@localhost:5432/barterswap?sslmode=disable"
	}

	st, err := store.New(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           api.NewServer(st).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("BarterSwap démarré sur http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
