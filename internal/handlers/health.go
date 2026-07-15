package handlers

import (
	"encoding/json"
	"net/http"
)

// Health répond au contrôle de santé utilisé par Docker et
// les outils d'observabilité (Uptime Kuma, Prometheus, ...).
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
