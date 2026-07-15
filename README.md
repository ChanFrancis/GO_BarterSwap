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

| Méthode | Path | Description |
|---------|------|-------------|
| GET | `/health` | État de l'API |

*(Le tableau sera complété au fil de l'implémentation : users, skills,
services, exchanges, reviews, stats — voir CLAUDE.md pour le plan.)*

## Exemples d'utilisation

```bash
curl http://localhost:8080/health
```

## Tests

```bash
go test -v -cover ./...
```
