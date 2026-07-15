package server

import "net/http"

// secureHeaders ajoute les en-têtes de sécurité recommandés (OWASP) sur
// toutes les réponses.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
