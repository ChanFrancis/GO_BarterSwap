package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
	"github.com/ChanFrancis/GO_BarterSwap/internal/store"
)

// Tests d'intégration sur une vraie base PostgreSQL. Ils se sautent
// automatiquement si TEST_DATABASE_URL n'est pas défini, de sorte que
// `go test` fonctionne sans base ; la CI les exécute contre un service
// PostgreSQL. Ils couvrent l'ensemble des couches (api → store → domaine).

type client struct {
	t *testing.T
	h http.Handler
}

func newTestClient(t *testing.T) *client {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL non défini : test d'intégration ignoré")
	}
	st, err := store.New(url)
	if err != nil {
		t.Fatalf("connexion à la base de test : %v", err)
	}
	if _, err := st.DB().Exec(
		`TRUNCATE reviews, credit_transactions, exchanges, services, skills, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("nettoyage de la base : %v", err)
	}
	return &client{t: t, h: NewServer(st).Routes()}
}

func (c *client) do(method, path, body string, userID int) *httptest.ResponseRecorder {
	c.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if userID > 0 {
		req.Header.Set("X-User-ID", strconv.Itoa(userID))
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec
}

func (c *client) mustStatus(rec *httptest.ResponseRecorder, want int, ctx string) {
	c.t.Helper()
	if rec.Code != want {
		c.t.Fatalf("%s : code %d attendu, reçu %d (%s)", ctx, want, rec.Code, rec.Body.String())
	}
}

func (c *client) balance(userID int) int {
	c.t.Helper()
	rec := c.do(http.MethodGet, "/api/users/"+strconv.Itoa(userID), "", 0)
	var u barterswap.User
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		c.t.Fatalf("décodage user : %v", err)
	}
	return u.CreditBalance
}

func TestIntegrationParcoursComplet(t *testing.T) {
	c := newTestClient(t)

	// Création des comptes : alice=1, bob=2, carol=3 (10 crédits chacun).
	for _, pseudo := range []string{"alice", "bob", "carol"} {
		rec := c.do(http.MethodPost, "/api/users", `{"pseudo":"`+pseudo+`"}`, 0)
		c.mustStatus(rec, http.StatusCreated, "création "+pseudo)
	}
	if bal := c.balance(1); bal != barterswap.CreditsBienvenue {
		t.Fatalf("crédits de bienvenue : attendu %d, reçu %d", barterswap.CreditsBienvenue, bal)
	}

	// bob déclare une compétence et publie deux services.
	c.mustStatus(c.do(http.MethodPut, "/api/users/2/skills",
		`[{"nom":"Cuisine","niveau":"expert"}]`, 2), http.StatusOK, "skills bob")
	c.mustStatus(c.do(http.MethodPost, "/api/services",
		`{"titre":"Cours de cuisine","categorie":"Cuisine","duree_minutes":90,"credits":3}`, 2),
		http.StatusCreated, "service 1")
	c.mustStatus(c.do(http.MethodPost, "/api/services",
		`{"titre":"Banquet","categorie":"Cuisine","duree_minutes":600,"credits":100}`, 2),
		http.StatusCreated, "service 2")

	// Publier sans compétence dans la catégorie → 400.
	c.mustStatus(c.do(http.MethodPost, "/api/services",
		`{"titre":"Dépannage","categorie":"Informatique","duree_minutes":30,"credits":1}`, 1),
		http.StatusBadRequest, "service sans compétence")

	// Règles de création d'échange.
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges", `{"service_id":1}`, 2),
		http.StatusBadRequest, "échange sur son propre service")
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges", `{"service_id":2}`, 1),
		http.StatusBadRequest, "crédits insuffisants")
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges", `{"service_id":1}`, 1),
		http.StatusCreated, "échange 1")
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges", `{"service_id":1}`, 3),
		http.StatusConflict, "service déjà réservé")

	// Compléter avant acceptation → transition invalide.
	c.mustStatus(c.do(http.MethodPut, "/api/exchanges/1/complete", "", 1),
		http.StatusConflict, "complete depuis pending")

	// Acceptation : crédits bloqués (alice 10→7, bob encore 10).
	c.mustStatus(c.do(http.MethodPut, "/api/exchanges/1/accept", "", 2),
		http.StatusOK, "accept")
	if got := c.balance(1); got != 7 {
		t.Fatalf("après accept, solde alice attendu 7, reçu %d", got)
	}
	if got := c.balance(2); got != 10 {
		t.Fatalf("après accept, solde bob attendu 10 (non crédité), reçu %d", got)
	}

	// Un tiers ne voit pas le détail de l'échange.
	c.mustStatus(c.do(http.MethodGet, "/api/exchanges/1", "", 3),
		http.StatusForbidden, "détail par un tiers")

	// Complétion : transfert des crédits (bob 10→13).
	c.mustStatus(c.do(http.MethodPut, "/api/exchanges/1/complete", "", 1),
		http.StatusOK, "complete")
	if got := c.balance(2); got != 13 {
		t.Fatalf("après complete, solde bob attendu 13, reçu %d", got)
	}

	// Évaluations : noter un échange terminé, une seule fois.
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges/1/review",
		`{"note":5,"commentaire":"Parfait"}`, 1), http.StatusCreated, "review alice")
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges/1/review",
		`{"note":3}`, 1), http.StatusBadRequest, "double review")
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges/1/review",
		`{"note":4}`, 2), http.StatusCreated, "review bob")

	// Cycle annulation : carol réserve puis annule, ses crédits reviennent.
	c.mustStatus(c.do(http.MethodPost, "/api/services",
		`{"titre":"Atelier pâtisserie","categorie":"Cuisine","duree_minutes":120,"credits":4}`, 2),
		http.StatusCreated, "service 3")
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges", `{"service_id":3}`, 3),
		http.StatusCreated, "échange 2")
	c.mustStatus(c.do(http.MethodPut, "/api/exchanges/2/accept", "", 2),
		http.StatusOK, "accept 2")
	if got := c.balance(3); got != 6 {
		t.Fatalf("après accept, solde carol attendu 6, reçu %d", got)
	}
	c.mustStatus(c.do(http.MethodPut, "/api/exchanges/2/cancel", "", 3),
		http.StatusOK, "cancel 2")
	if got := c.balance(3); got != barterswap.CreditsBienvenue {
		t.Fatalf("après cancel, solde carol restitué à %d, reçu %d", barterswap.CreditsBienvenue, got)
	}

	// Refus d'une demande en attente (aucun crédit bloqué).
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges", `{"service_id":3}`, 1),
		http.StatusCreated, "échange 3")
	c.mustStatus(c.do(http.MethodPut, "/api/exchanges/3/reject", "", 2),
		http.StatusOK, "reject")

	// Noter un échange non terminé → 400.
	c.mustStatus(c.do(http.MethodPost, "/api/exchanges/3/review", `{"note":5}`, 1),
		http.StatusBadRequest, "review échange non terminé")

	// Statistiques cohérentes pour bob.
	rec := c.do(http.MethodGet, "/api/users/2/stats", "", 0)
	c.mustStatus(rec, http.StatusOK, "stats bob")
	var stats barterswap.UserStats
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("décodage stats : %v", err)
	}
	if stats.EchangesCompletes != 1 || stats.NbAvis != 1 || stats.NoteMoyenne != 5 {
		t.Fatalf("stats bob incohérentes : %+v", stats)
	}
	if stats.CreditBalance != stats.TotalGagne-stats.TotalDepense {
		t.Fatalf("stats bob : solde %d != gagné %d - dépensé %d",
			stats.CreditBalance, stats.TotalGagne, stats.TotalDepense)
	}

	// Listes filtrées et avis.
	c.mustStatus(c.do(http.MethodGet, "/api/exchanges?status=completed", "", 1),
		http.StatusOK, "liste échanges complétés")
	c.mustStatus(c.do(http.MethodGet, "/api/users/2/reviews", "", 0),
		http.StatusOK, "avis reçus par bob")
	c.mustStatus(c.do(http.MethodGet, "/api/services/1/reviews", "", 0),
		http.StatusOK, "avis du service 1")
}
