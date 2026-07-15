# CLAUDE.md — BarterSwap

## Contexte

BarterSwap est une plateforme de troc (échange d'objets sans monnaie) réalisée
pour le Projet Annuel ESGI (groupe de 3-4 personnes). Backend en Go, base
PostgreSQL, tout tourne via Docker — **Go n'est pas installé en local**.

## Commandes

```bash
# Lancer l'application (API + PostgreSQL)
cp .env.example .env && docker compose up --build

# Tests / vet / build (toujours via Docker)
docker run --rm -v "$PWD":/app -w /app golang:1.26 go test ./...
docker run --rm -v "$PWD":/app -w /app golang:1.26 go vet ./...
docker run --rm -v "$PWD":/app -w /app golang:1.26 go build -buildvcs=false ./...
```

## Architecture

```
cmd/server/        Point d'entrée
internal/config/   Configuration via variables d'environnement (.env non commité)
internal/server/   Construction du serveur HTTP et routage (net/http, Go 1.22+ patterns)
internal/handlers/ Handlers HTTP, un fichier par domaine + test associé
```

Conventions : bibliothèque standard en priorité, dépendances minimales et
justifiées, code et messages en français pour la documentation, chaque handler
accompagné de son test `_test.go`.

---

# Plan pour obtenir tous les points du sujet

## Phase 1 — Authentification & Sécurité (section « Sécurité »)

1. **Modèle utilisateur + migrations** : table `users` (email unique, hash
   argon2id, dates de création/changement de mot de passe), migrations SQL
   versionnées (golang-migrate).
2. **Inscription** (`POST /api/register`) : validation email, mot de passe
   fort obligatoire — **12 caractères minimum avec chiffres, lettres et
   symboles** (règle CNIL), hash argon2id, jamais de mot de passe en clair.
3. **Connexion** (`POST /api/login`) : sessions via cookie `HttpOnly`,
   `Secure`, `SameSite=Strict` (ou JWT courte durée + refresh token).
4. **Blocage des tentatives infructueuses** (règle CNIL) : compteur d'échecs
   par compte + IP, verrouillage temporaire progressif après N échecs,
   journalisation des tentatives.
5. **Mot de passe oublié / réinitialisation** : token à usage unique, haché en
   base, expiration courte (15-30 min), envoi par email (SMTP conteneurisé —
   Mailpit en dev), réponse identique que l'email existe ou non.
6. **Expiration du mot de passe à 60 jours** (règle CNIL) : à la connexion,
   si `password_changed_at` > 60 jours → forcer la réinitialisation.
7. **Durcissement transversal** : middleware de sécurité (en-têtes CSP,
   X-Content-Type-Options, HSTS), rate limiting global, validation stricte
   des entrées, requêtes SQL paramétrées uniquement.

## Phase 2 — Métier (section « Réponse Métier et Architecture »)

1. **Objets** : CRUD des objets à troquer (titre, description, catégorie,
   état, photos uploadées), objets liés à leur propriétaire.
2. **Offres de troc** : proposer un échange (mes objets X contre ton objet Y),
   accepter / refuser / annuler, historique des échanges.
3. **Recherche & catalogue** : liste paginée, filtres par catégorie, recherche
   texte.
4. **Messagerie entre troqueurs** (justifie le WebSocket demandé en infra) :
   conversation par offre de troc.
5. Architecture claire par domaine (`internal/items`, `internal/trades`,
   `internal/auth`...) : pas de Clean Architecture imposée, mais séparation
   handlers / logique métier / accès données, maintenable et lisible.

## Phase 3 — Frontend, Design & Accessibilité

1. **Client web** (framework au choix du groupe, ou templates Go + HTMX) :
   parcours complet inscription → dépôt d'objet → offre → échange.
2. **Accessibilité** : HTML sémantique, labels sur tous les champs, contraste
   suffisant, navigation clavier, attributs ARIA — viser un audit Lighthouse
   accessibilité > 90.
3. **SEO / robots** : balises meta, sitemap.xml, robots.txt, URLs propres.
4. **UX** : messages d'erreur clairs, états de chargement, responsive.

## Phase 4 — Tests (section « Tests »)

1. **Unitaires** : logique métier (validation mot de passe, règles de troc),
   handlers avec `httptest` — déjà démarré avec `health_test.go`.
2. **Fonctionnels** : tests d'intégration API avec base PostgreSQL éphémère
   (testcontainers-go ou compose de test) couvrant les parcours complets
   (inscription → connexion → troc).
3. **Interface** : tests end-to-end avec Playwright sur les parcours
   critiques.
4. **CI GitHub Actions** : `go vet` + tests + build Docker à chaque push —
   prouve la régularité et la qualité au correcteur.

## Phase 5 — Infrastructure (section « Infrastructure »)

1. **VPS** (Hetzner, OVH, Scaleway...) avec Docker installé via IaC.
2. **Reverse proxy** : Caddy ou Traefik en conteneur — domaine public +
   **certificat SSL Let's Encrypt automatique** (autorité de confiance).
3. **Pare-feu** : UFW ou nftables, seuls 22/80/443 ouverts, SSH par clé
   uniquement — configuré par IaC pour le prouver.
4. **Registre Docker** : GitHub Container Registry (ghcr.io) — les images
   sont buildées en CI puis tirées par le VPS.
5. **WebSocket** : servi par l'API Go pour la messagerie (Phase 2), passe par
   le reverse proxy.
6. **IaC obligatoire** : Dockerfile (fait) + Ansible (provisioning VPS :
   pare-feu, Docker, déploiement) ou Terraform (création du VPS). Tout doit
   être reproductible depuis le repo.

## Phase 6 — Observabilité (section « Observabilité »)

1. **Santé des conteneurs** : Uptime Kuma en conteneur qui surveille
   `/health` de l'API + la base (le endpoint existe déjà).
2. **Erreurs** : GlitchTip auto-hébergé (compatible SDK Sentry, gratuit) ;
   intégrer le SDK Sentry-Go dans l'API.
3. **Analytique** : Plausible auto-hébergé sur le client web (léger, RGPD).

## Phase 7 — Sauvegardes 3-2-1 (section « Politique de recouvrement »)

1. `pg_dump` quotidien via conteneur cron + sauvegarde des fichiers uploadés.
2. **3 copies, 2 médiums, 1 externe** : (1) disque du VPS, (2) Volume/Block
   Storage du provider, (3) bucket S3 externe chez un autre cloud
   (Backblaze B2, AWS...) — chiffrées (restic ou rclone + age).
3. **Tester la restauration** et documenter la procédure dans `docs/`.

## Phase 8 — Gestion de projet (notée à chaque séance de suivi)

1. **GitHub Projects** : backlog reprenant ce plan, colonnes To do / In
   progress / Done, issues assignées.
2. **Répartition équitable** : chaque membre a des issues à son nom, travail
   via branches + Pull Requests reviewées.
3. **Régularité** : commits fréquents de tout le monde (l'historique `.git`
   est livré et audité), pas de gros dump la veille de la séance.

## Phase 9 — Bonus (points supplémentaires)

- **Authentification avancée** : OAuth2/OIDC Google + lien magique par email,
  et **2FA TOTP** (compatible Google/Microsoft Authenticator) — bonus le plus
  rentable, s'appuie sur la Phase 1.
- **RGPD** : pages CGU/CGV/Contact, bannière cookies, politique de
  confidentialité, export/suppression des données du compte.
- **Application mobile** : client Capacitor réutilisant le front web
  (obligation : app native installable ; store facultatif).
- **Auto-hébergement/réplication** (si matériel disponible) : Proxmox + k3s
  ou Docker Swarm, 2 répliques par service, CloudFlare Tunnel.

## Livrables finaux (Contraintes)

- Archive du repo **avec le dossier `.git`** (contributions individuelles
  visibles) uploadée sur MyGES.
- Documentation permettant de reproduire le projet en local (README) **et**
  l'infrastructure (IaC + docs/).
- Code propre : `go vet` sans erreur, nommage cohérent, fonctions courtes.

## Ordre de travail recommandé

Phase 1 → 2 → 4 (en continu dès maintenant) → 3 → 5 → 6 → 7, avec la Phase 8
(gestion de projet) dès aujourd'hui et les bonus en fin de parcours. Mettre en
production (Phase 5) tôt, même avec peu de fonctionnalités : le déploiement
continu prouve la régularité et évite les mauvaises surprises de la dernière
semaine.
