// BarterSwap est une API d'échange de compétences entre particuliers :
// chaque heure de service rendue donne droit à une heure de service reçue,
// comptée en crédits-temps.
package main

import (
	"log"
	"net/http"
	"os"
	"time"
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

	db, err := openDB(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	a := &app{db: db}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           a.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("BarterSwap démarré sur http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
