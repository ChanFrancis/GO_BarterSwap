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

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           newRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("BarterSwap démarré sur http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
