package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

// withMiddlewares enchaîne recovery → CORS → logging sur tout le routeur.
func withMiddlewares(next http.Handler) http.Handler {
	return recovery(cors(logging(next)))
}

// logging trace chaque requête (méthode, chemin, durée).
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(debut))
	})
}

// recovery transforme une panique en 500 au lieu de tuer le serveur.
func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panique récupérée : %v", err)
				writeError(w, http.StatusInternalServerError, "erreur interne")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// cors autorise les appels depuis un navigateur (front en localhost).
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// currentUserID lit l'utilisateur courant dans le header X-User-ID
// (le sujet n'exige pas d'authentification plus avancée).
func currentUserID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.Header.Get("X-User-ID"))
	if err != nil || id <= 0 {
		return 0, barterswap.ValidationError{Message: "header X-User-ID manquant ou invalide"}
	}
	return id, nil
}

// pathID lit l'identifiant numérique {id} du chemin.
func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, barterswap.ValidationError{Message: "identifiant invalide dans l'URL"}
	}
	return id, nil
}

// idAndCaller lit à la fois l'identifiant {id} du chemin et l'appelant
// (X-User-ID) : combinaison utilisée par les routes réservées au
// propriétaire d'une ressource.
func idAndCaller(r *http.Request) (id, callerID int, err error) {
	if id, err = pathID(r); err != nil {
		return 0, 0, err
	}
	if callerID, err = currentUserID(r); err != nil {
		return 0, 0, err
	}
	return id, callerID, nil
}
