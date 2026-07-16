package api

import (
	"embed"
	"io/fs"
	"net/http"
)

// La spec OpenAPI et l'interface Swagger UI sont embarquées dans le binaire
// (aucune dépendance Go, aucun accès réseau au runtime). Swagger UI est un
// simple ensemble de fichiers statiques servis par net/http.

//go:embed openapi.yaml
var openAPISpec []byte

//go:embed swaggerui
var swaggerFiles embed.FS

// handleOpenAPISpec sert la spécification OpenAPI.
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openAPISpec)
}

// swaggerHandler sert l'interface Swagger UI sous /docs/.
func swaggerHandler() http.Handler {
	sub, err := fs.Sub(swaggerFiles, "swaggerui")
	if err != nil {
		panic(err) // impossible : le dossier est embarqué à la compilation
	}
	return http.StripPrefix("/docs/", http.FileServer(http.FS(sub)))
}
