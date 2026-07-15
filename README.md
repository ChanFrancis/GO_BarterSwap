# GO_BarterSwap

BarterSwap est une plateforme de troc : les utilisateurs proposent des objets
et s'échangent des offres, sans monnaie. Projet annuel ESGI (backend en Go).

## Prérequis

- Docker et Docker Compose (Go n'est pas nécessaire en local)

## Lancer le projet

```bash
cp .env.example .env   # puis renseigner POSTGRES_PASSWORD
docker compose up --build
```

L'API répond ensuite sur http://localhost:8080/health

## Lancer les tests

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.26 go test ./...
```

## Structure du projet

```
cmd/server/        Point d'entrée de l'application
internal/config/   Chargement de la configuration (variables d'environnement)
internal/server/   Construction du serveur HTTP et des routes
internal/handlers/ Handlers HTTP (un fichier par domaine)
```

## Feuille de route (exigences du sujet)

- [x] Squelette Go + Docker + PostgreSQL
- [ ] Authentification : inscription, connexion, mot de passe oublié/réinitialisation
- [ ] Sécurité CNIL : mot de passe fort (12+ caractères), expiration 60 jours, blocage après échecs
- [ ] 2FA (TOTP) et OAuth2 (bonus)
- [ ] Métier : objets, offres de troc, échanges
- [ ] Tests unitaires, fonctionnels et d'interface
- [ ] Observabilité : santé des conteneurs, erreurs (Sentry/GlitchTip), analytique
- [ ] Déploiement VPS : registre Docker, pare-feu, domaine + SSL
- [ ] Sauvegardes 3-2-1 de la base de données
- [ ] Pages légales (CGU, contact, cookies)
