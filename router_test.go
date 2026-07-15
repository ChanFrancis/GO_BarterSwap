package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Tests d'API sans base de données : routes et chemins d'erreur qui
// n'atteignent pas le stockage. Les parcours complets sont vérifiés via
// docker compose (voir README).

func doRequest(t *testing.T, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	(&app{}).routes().ServeHTTP(rec, req)
	return rec
}

func TestRoutes(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		headers    map[string]string
		wantStatus int
	}{
		{"health", http.MethodGet, "/health", "", nil, http.StatusOK},
		{"création avec pseudo vide → 400", http.MethodPost, "/api/users",
			`{"pseudo":""}`, nil, http.StatusBadRequest},
		{"création avec JSON invalide → 400", http.MethodPost, "/api/users",
			`{pseudo`, nil, http.StatusBadRequest},
		{"profil avec id non numérique → 400", http.MethodGet, "/api/users/abc",
			"", nil, http.StatusBadRequest},
		{"modification sans X-User-ID → 400", http.MethodPut, "/api/users/1",
			`{"pseudo":"francis"}`, nil, http.StatusBadRequest},
		{"modification du profil d'un autre → 403", http.MethodPut, "/api/users/1",
			`{"pseudo":"francis"}`, map[string]string{"X-User-ID": "2"}, http.StatusForbidden},
		{"skills d'un autre → 403", http.MethodPut, "/api/users/5/skills",
			`[]`, map[string]string{"X-User-ID": "3"}, http.StatusForbidden},
		{"méthode inconnue sur users → 405", http.MethodDelete, "/api/users/1",
			"", nil, http.StatusMethodNotAllowed},
		{"preflight CORS → 204", http.MethodOptions, "/api/users",
			"", nil, http.StatusNoContent},
		{"créer un service sans X-User-ID → 400", http.MethodPost, "/api/services",
			`{"titre":"x","categorie":"Musique","duree_minutes":60,"credits":2}`, nil, http.StatusBadRequest},
		{"créer un service catégorie invalide → 400", http.MethodPost, "/api/services",
			`{"titre":"x","categorie":"Astrologie","duree_minutes":60,"credits":2}`,
			map[string]string{"X-User-ID": "1"}, http.StatusBadRequest},
		{"service id non numérique → 400", http.MethodGet, "/api/services/abc",
			"", nil, http.StatusBadRequest},
		{"supprimer un service sans X-User-ID → 400", http.MethodDelete, "/api/services/1",
			"", nil, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := doRequest(t, c.method, c.path, c.body, c.headers)
			if rec.Code != c.wantStatus {
				t.Errorf("code attendu %d, reçu %d (corps : %s)",
					c.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHealthBody(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/health", "", nil)
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("corps inattendu : %s", rec.Body.String())
	}
}
