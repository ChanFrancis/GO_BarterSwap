# CLAUDE.md — BarterSwap (Projet de fin de module Go, Sujet-7)

## Contexte

BarterSwap est une **API d'échange de compétences entre particuliers** : une
banque de temps où chaque heure de service rendue donne droit à une heure de
service reçue, comptée en **crédits-temps**. Ce n'est ni une plateforme de
freelance, ni du troc direct : les échanges sont différés via les crédits.

Projet noté sur 20, soutenance de 10 min (6 min de démo curl en direct +
4 min de questions/test de résilience). Groupe de 3. Le sujet complet est
dans `../Sujet-7.pdf`.

**Attention : ne pas confondre avec `Sujet-6.pdf` (Projet Annuel), qui est un
autre cours.** Ici : pas de VPS, pas de SSL, pas d'authentification, pas
d'infra de prod.

## Contraintes STRICTES du sujet (éliminatoires si violées)

- **Structure Go idiomatique** : `cmd/` + `internal/` multi-packages (voir
  Architecture). Le professeur a précisé : pas de package fourre-tout à la
  racine, pas l'interdiction des sous-packages.
- **Une seule dépendance externe autorisée : le driver de base de données**
  (lib/pq, pgx, go-sql-driver/mysql…). Rien d'autre, pas de x/crypto.
- **Pas d'ORM** : `database/sql` de la stdlib uniquement.
- **Pas de framework HTTP** (ni Gin, ni Echo, ni Chi) : `net/http` seul.
- **Pas de mutex** : la base de données gère la concurrence.
- **Pas d'authentification avancée** : l'utilisateur courant est identifié
  par le header `X-User-ID`.
- Filtres et recherche **côté serveur** (query parameters).

## Commandes

Go n'est pas installé sur la machine : tout passe par Docker.

```bash
# Base de données + API
cp .env.example .env && docker compose up --build

# Tests, vet, format, couverture (exigence : ≥ 60 %)
docker run --rm -u "$(id -u):$(id -g)" -e GOCACHE=/tmp/gocache -e GOPATH=/tmp/gopath \
  -v "$PWD":/app -w /app golang:1.26 go test -v -cover ./...
docker run --rm -v "$PWD":/app -w /app golang:1.26 go vet ./...
docker run --rm -v "$PWD":/app -w /app golang:1.26 gofmt -l .
```

## Architecture (3 points au barème)

Structure idiomatique Go (Module 8 du cours) : `cmd/` pour l'exécutable,
`internal/` pour le code applicatif privé, découpé en **trois packages par
couche** avec un sens de dépendance unique et sans cycle :

```
cmd/barterswap ──▶ api ──▶ store ──▶ barterswap (domaine)
                    └───────────────▶ barterswap
```

```
cmd/barterswap/main.go        Point d'entrée : config, câblage, démarrage

internal/barterswap/          DOMAINE — aucune dépendance externe
  models.go                     entités (tags JSON imposés)
  errors.go                     sentinelles + ValidationError
  rules.go                      validations pures, statuts, cycle de vie
  rules_test.go                 tests table-driven (sans base)

internal/store/               ACCÈS DONNÉES (PostgreSQL)
  store.go                      type Store, New, Close, DB, schéma embarqué
  users.go services.go exchanges.go reviews.go
  schema.sql

internal/api/                 EXPOSITION HTTP
  server.go                     type Server, Routes, helpers JSON/erreurs
  middleware.go                 X-User-ID, logging, recovery, CORS
  users.go services.go exchanges.go reviews.go
  server_test.go                httptest (sans base)
  integration_test.go           parcours complet sur vraie base

scripts/demo.sh               Démonstration des 12 cas métier (soutenance)
```

Le **domaine** ne connaît ni HTTP ni SQL ; le **store** le remplit depuis
PostgreSQL et renvoie des types du domaine ; l'**api** appelle le store et
parle HTTP. La séparation des couches est ainsi **structurelle** (garantie
par le compilateur), pas seulement une convention de nommage.

Conventions (Module 8) : stdlib uniquement, visibilité par la casse (exporté
= majuscule), commentaires godoc commençant par le nom de l'identifiant,
erreurs sentinelles (`ErrX = errors.New`) + `errors.Is`/`As`, wrapping `%w`,
gofmt obligatoire, code et messages en français.

**Historique** — le sujet disait « un seul package Go » ; le professeur a
précisé qu'il s'agissait de ne pas laisser un package fourre-tout à la racine.
On applique donc la structure `cmd/`+`internal/` multi-packages du cours.

## Règles métier des crédits (cœur de la notation « Fonctionnalités »)

- Création de compte → **10 crédits de bienvenue**.
- Les crédits sont un **journal de transactions** (`credit_transactions` :
  montant positif = crédit, négatif = débit ; type `earn`/`spend`/`refund`),
  le solde est la somme du journal — pas un simple champ.
- `POST /api/exchanges` : refusé si le demandeur n'a pas assez de crédits
  (400), si le service est le sien (400), ou si le service a déjà un échange
  `pending`/`accepted` (409).
- `accepted` → crédits **bloqués** (débités du demandeur, pas encore crédités
  à l'offreur).
- `completed` → crédits **transférés** à l'offreur.
- `rejected`/`cancelled` → crédits **restitués** au demandeur.
- Cycle de vie : `pending → accepted → completed`, avec `rejected` depuis
  pending et `cancelled` depuis accepted (demandeur ou offreur).
- Reviews : uniquement sur un échange `completed`, 1 seul avis par
  utilisateur et par échange (400 sinon), note 1-5, ni modifiable ni
  supprimable.

## Endpoints imposés

- Users : `POST /api/users`, `GET|PUT /api/users/{id}`,
  `GET|PUT /api/users/{id}/skills` (PUT écrase toutes les skills)
- Services : CRUD `/api/services` + filtres `?categorie=`, `?ville=`,
  `?search=` (catégories : liste fermée de 13 valeurs, voir sujet p.4)
- Exchanges : `POST|GET /api/exchanges`, `GET /api/exchanges/{id}`,
  `PUT /api/exchanges/{id}/accept|reject|complete|cancel`, `?status=`
- Reviews : `POST /api/exchanges/{id}/review`, `GET /api/users/{id}/reviews`,
  `GET /api/services/{id}/reviews`
- Stats : `GET /api/users/{id}/stats` (UserStats complet)

## État d'avancement

Toutes les fonctionnalités du sujet sont implémentées : users/skills,
services (filtres serveur), exchanges (cycle de vie + crédits en journal de
transactions), reviews, stats. Tests unitaires (validations, routage) +
test d'intégration du parcours complet sur vraie base (skip si
`TEST_DATABASE_URL` absent). Couverture ~64 %, CI avec PostgreSQL et seuil
à 60 %.

Reste surtout : préparation de la soutenance (script de démo curl couvrant
les 12 cas), relecture qualité, éventuels bonus jury.

## Plan de travail (aligné sur le barème /20)

1. **Socle** : schéma SQL, connexion `database/sql`, middlewares (X-User-ID,
   logging, recovery, CORS), helpers JSON/erreurs. [Architecture 3 pts]
2. **Users + skills** : création avec 10 crédits, profil, PUT skills qui
   écrase. Premier jeu de tests table-driven + httptest. [Fonctionnalités]
3. **Services** : CRUD (propriétaire uniquement), compétence requise pour
   publier (400 sinon), filtres serveur categorie/ville/search.
4. **Exchanges + crédits** : journal de transactions, cycle de vie complet,
   toutes les règles ci-dessus **dans business.go**, transactions SQL pour
   accept/complete/cancel. C'est le morceau le plus noté et le plus testé en
   soutenance.
5. **Reviews + stats** : contraintes d'unicité en base, agrégats SQL pour
   UserStats.
6. **Tests jusqu'à ≥ 60 % de couverture** [3 pts] : table-driven sur
   business.go, httptest sur chaque endpoint, et les 12 cas métier listés
   dans le sujet (p.9-10) comme checklist minimale.
7. **Gestion d'erreurs** [1 pt] : sentinelles → codes HTTP cohérents
   (400 validation, 403 pas le propriétaire, 404 introuvable, 409 conflit de
   réservation), messages JSON clairs.
8. **README** [1 pt] : format imposé par le sujet — Installation
   (`go mod tidy && go run .`), tableau des endpoints, 3-4 exemples curl
   complets, section tests (`go test -v -cover ./...`).
9. **Préparation soutenance** : script de démo curl (cas nominaux + cas
   d'erreur), chacun des 3 membres sait expliquer chaque couche. [Bonus jury
   5 pts : originalité, dépassement — la CI GitHub Actions existante y
   contribue]

## Les 12 cas métier du sujet (checklist de démo et de tests)

1. Créer un utilisateur → 201 · 2. Pseudo vide → 400 · 3. Publier un service
sans avoir la compétence → 400 · 4. Échange sur son propre service → 400 ·
5. Échange sans crédits suffisants → 400 · 6. Échange sur un service déjà
réservé → 409 · 7. Accepter → crédits bloqués, statut `accepted` ·
8. Compléter → crédits transférés, statut `completed` · 9. Annuler → crédits
restitués · 10. Noter un échange non terminé → 400 · 11. Noter deux fois →
400 · 12. Stats → valeurs cohérentes.

## Rendu

- Dépôt Git complet **avec l'historique** (dossier `.git` inclus).
- Jamais d'attribution Claude dans les commits.
- Contributions des 3 membres visibles dans l'historique.
