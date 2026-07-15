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

*(À venir : exchanges, reviews, stats — voir CLAUDE.md.)*

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
```

## Tests

```bash
go test -v -cover ./...
```
