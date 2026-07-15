# BarterSwap — API d'échange de compétences

Plateforme d'échange de compétences entre particuliers, fonctionnant comme
une **banque de temps** : chaque heure de service rendue donne droit à une
heure de service reçue, comptée en crédits-temps. API REST en Go (stdlib
uniquement) avec PostgreSQL.

## Installation

```bash
git clone git@github.com:ChanFrancis/GO_BarterSwap.git
cd GO_BarterSwap
go mod tidy
go run .
```

Avec Docker (recommandé, lance aussi PostgreSQL) :

```bash
cp .env.example .env   # renseigner POSTGRES_PASSWORD
docker compose up --build
```

## Endpoints

L'utilisateur courant est identifié par le header `X-User-ID` (pas
d'authentification avancée, conformément au sujet).

| Méthode | Path | Description |
|---------|------|-------------|
| GET | `/health` | État de l'API |
| POST | `/api/users` | Créer un compte (10 crédits de bienvenue) |
| GET | `/api/users/{id}` | Profil public (avec compétences et solde) |
| PUT | `/api/users/{id}` | Modifier son profil |
| GET | `/api/users/{id}/skills` | Compétences d'un utilisateur |
| PUT | `/api/users/{id}/skills` | Définir ses compétences (écrase la liste) |
| GET | `/api/services` | Annonces actives (filtres `?categorie=`, `?ville=`, `?search=`) |
| POST | `/api/services` | Publier une annonce (compétence requise dans la catégorie) |
| GET | `/api/services/{id}` | Détail d'une annonce |
| PUT | `/api/services/{id}` | Modifier son annonce |
| DELETE | `/api/services/{id}` | Supprimer son annonce |
| POST | `/api/exchanges` | Demander un échange (`{"service_id":N}`) |
| GET | `/api/exchanges` | Ses échanges (demandés + reçus, filtre `?status=`) |
| GET | `/api/exchanges/{id}` | Détail d'un échange |
| PUT | `/api/exchanges/{id}/accept` | Accepter (offreur) : bloque les crédits |
| PUT | `/api/exchanges/{id}/reject` | Refuser (offreur) une demande en attente |
| PUT | `/api/exchanges/{id}/complete` | Terminer : transfère les crédits à l'offreur |
| PUT | `/api/exchanges/{id}/cancel` | Annuler : restitue les crédits bloqués |
| POST | `/api/exchanges/{id}/review` | Noter un échange terminé (`{"note":1-5,"commentaire":"…"}`) |
| GET | `/api/users/{id}/reviews` | Avis reçus par un utilisateur |
| GET | `/api/services/{id}/reviews` | Avis portant sur un service |
| GET | `/api/users/{id}/stats` | Tableau de bord d'un utilisateur |

Un avis n'est possible que sur un échange terminé, une seule fois par
utilisateur et par échange, et ne peut être ni modifié ni supprimé.

### Crédits-temps et cycle de vie d'un échange

```
pending ──accept──► accepted ──complete──► completed
   │                    │
 reject               cancel
   ▼                    ▼
rejected            cancelled
```

- Création d'un compte → 10 crédits de bienvenue.
- `accept` : les crédits du demandeur sont **bloqués** (débités, pas encore
  versés à l'offreur).
- `complete` : les crédits sont **transférés** à l'offreur.
- `cancel` / `reject` : les crédits bloqués sont **restitués** au demandeur.
- Le solde est le cumul d'un journal de transactions (`earn`/`spend`/`refund`),
  jamais un champ stocké.

Catégories acceptées : Informatique, Jardinage, Bricolage, Cuisine, Musique,
Langues, Sport, Tutorat, Déménagement, Photographie, Animalier, Couture, Autre.
On ne peut publier un service que dans une catégorie où l'on a déclaré une
compétence de même nom.

## Exemples d'utilisation

```bash
# Créer un compte (10 crédits de bienvenue attribués automatiquement)
curl -X POST http://localhost:8080/api/users \
  -d '{"pseudo":"alice","bio":"Jardinière du dimanche","ville":"Paris"}'

# Définir ses compétences (niveaux : débutant, intermédiaire, expert)
curl -X PUT http://localhost:8080/api/users/1/skills \
  -H "X-User-ID: 1" \
  -d '[{"nom":"Jardinage","niveau":"expert"},{"nom":"Cuisine","niveau":"débutant"}]'

# Consulter un profil (compétences + solde de crédits)
curl http://localhost:8080/api/users/1

# Publier un service (nécessite une compétence dans la catégorie)
curl -X POST http://localhost:8080/api/services \
  -H "X-User-ID: 1" \
  -d '{"titre":"Cours de piano","categorie":"Musique","duree_minutes":60,"credits":2,"ville":"Lyon"}'

# Rechercher des services
curl "http://localhost:8080/api/services?categorie=Musique&search=piano"

# Demander un échange, puis l'offreur l'accepte (crédits bloqués)
curl -X POST http://localhost:8080/api/exchanges -H "X-User-ID: 2" -d '{"service_id":1}'
curl -X PUT http://localhost:8080/api/exchanges/1/accept -H "X-User-ID: 1"

# Terminer l'échange (crédits transférés à l'offreur)
curl -X PUT http://localhost:8080/api/exchanges/1/complete -H "X-User-ID: 2"
```

## Démonstration

Le script [`demo.sh`](demo.sh) déroule les 12 cas métier du sujet (cas
nominaux et cas d'erreur) et affiche chaque code HTTP. Sur une base fraîche :

```bash
docker compose down -v && docker compose up --build -d
./demo.sh
```

## Tests

Tests unitaires (validations, routage) sans base :

```bash
go test -v -cover ./...
```

Tests d'intégration (parcours complet sur une vraie base) : ils se sautent
si `TEST_DATABASE_URL` n'est pas défini. Exemple avec Docker :

```bash
docker run -d --name pg -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=test -p 55432:5432 postgres:17
TEST_DATABASE_URL="postgres://test:test@localhost:55432/test?sslmode=disable" \
  go test -cover ./...
```

La CI GitHub Actions exécute ces tests contre un service PostgreSQL et
échoue si la couverture passe sous 60 %.
