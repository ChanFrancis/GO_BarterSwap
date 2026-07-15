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

L'API répond ensuite sur http://localhost:8080/health et les emails de
développement sont consultables sur http://localhost:8025 (Mailpit).

## API d'authentification

| Méthode | Route                  | Description                                    |
|---------|------------------------|------------------------------------------------|
| POST    | `/api/register`        | Inscription (`{"email", "password"}`)          |
| POST    | `/api/login`           | Connexion, pose un cookie de session           |
| POST    | `/api/logout`          | Déconnexion                                    |
| POST    | `/api/password/forgot` | Demande de réinitialisation (`{"email"}`)      |
| POST    | `/api/password/reset`  | Réinitialisation (`{"token", "new_password"}`) |
| GET     | `/api/me`              | Route protégée (session requise)               |

Règles CNIL appliquées : mot de passe de 12 caractères minimum avec lettres,
chiffres et symboles ; blocage du compte 15 minutes après 5 échecs de
connexion ; réinitialisation forcée si le mot de passe a plus de 60 jours.

## API métier (troc)

| Méthode | Route                      | Description                                              |
|---------|----------------------------|----------------------------------------------------------|
| GET     | `/api/items`               | Catalogue public (`?q=`, `?category=`, `?owner_id=`, `?page=`) |
| GET     | `/api/items/{id}`          | Détail d'un objet                                        |
| POST    | `/api/items` 🔒            | Créer un objet (`{"title","description","category","condition"}`) |
| PUT     | `/api/items/{id}` 🔒       | Modifier son objet                                       |
| DELETE  | `/api/items/{id}` 🔒       | Supprimer son objet                                      |
| POST    | `/api/trades` 🔒           | Proposer un troc (`{"requested_item_id","offered_item_ids","message"}`) |
| GET     | `/api/trades` 🔒           | Ses offres envoyées et reçues                            |
| POST    | `/api/trades/{id}/accept` 🔒 | Accepter (propriétaire de l'objet demandé) : échange la propriété |
| POST    | `/api/trades/{id}/decline` 🔒 | Refuser une offre reçue                                |
| POST    | `/api/trades/{id}/cancel` 🔒 | Annuler une offre envoyée                               |

🔒 = session requise. États d'objet acceptés : `neuf`, `très bon`, `bon`, `usé`.
À l'acceptation d'un troc, la propriété des objets est échangée en transaction
et les autres offres en attente sur ces objets sont automatiquement refusées.

## Lancer les tests

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.26 go test ./...
```

## Structure du projet

```
cmd/server/        Point d'entrée de l'application
internal/config/   Chargement de la configuration (variables d'environnement)
internal/server/   Construction du serveur HTTP, routes et middlewares
internal/handlers/ Handlers HTTP (un fichier par domaine)
internal/auth/     Logique d'authentification (hash argon2id, sessions, règles CNIL)
internal/items/    Objets à troquer : validation, CRUD, recherche
internal/trades/   Offres de troc : proposition, acceptation transactionnelle
internal/database/ Connexion PostgreSQL et migrations SQL embarquées
internal/mailer/   Envoi d'emails via SMTP
```

## Feuille de route (exigences du sujet)

- [x] Squelette Go + Docker + PostgreSQL
- [x] Authentification : inscription, connexion, mot de passe oublié/réinitialisation
- [x] Sécurité CNIL : mot de passe fort (12+ caractères), expiration 60 jours, blocage après échecs
- [ ] 2FA (TOTP) et OAuth2 (bonus)
- [x] Métier : objets, offres de troc, échanges (reste : photos, messagerie)
- [ ] Tests unitaires, fonctionnels et d'interface
- [ ] Observabilité : santé des conteneurs, erreurs (Sentry/GlitchTip), analytique
- [ ] Déploiement VPS : registre Docker, pare-feu, domaine + SSL
- [ ] Sauvegardes 3-2-1 de la base de données
- [ ] Pages légales (CGU, contact, cookies)
