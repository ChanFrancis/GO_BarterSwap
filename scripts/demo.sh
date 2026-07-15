#!/usr/bin/env bash
#
# Script de démonstration pour la soutenance BarterSwap.
# Déroule les 12 cas métier du sujet (cas nominaux + cas d'erreur).
#
# Prérequis : une base FRAÎCHE. Lancer avant :
#   docker compose down -v && docker compose up --build -d
#
# Usage : ./demo.sh   (ou BASE=http://mon-hote:8080 ./demo.sh)

set -u
BASE="${BASE:-http://localhost:8080}"

# req MÉTHODE CHEMIN [CORPS_JSON] [X-USER-ID]
req() {
	local method=$1 path=$2 body=${3:-} uid=${4:-}
	local args=(-s -w $'\n→ HTTP %{http_code}\n' -X "$method" "$BASE$path")
	[ -n "$body" ] && args+=(-d "$body")
	[ -n "$uid" ] && args+=(-H "X-User-ID: $uid")
	echo "\$ curl -X $method $path ${uid:+(X-User-ID: $uid)} ${body:+-d '$body'}"
	curl "${args[@]}"
	echo
}

titre() { echo; echo "=============================================================="; echo "$1"; echo "=============================================================="; }
solde() { echo "   solde de $1 : $(curl -s "$BASE/api/users/$2" | grep -o '"credit_balance":[0-9]*' | cut -d: -f2) crédits"; }

titre "PRÉPARATION — comptes, compétence et annonces"
# Cas 1 : création d'utilisateurs (10 crédits de bienvenue chacun)
echo "[CAS 1] Créer un utilisateur → 201"
req POST /api/users '{"pseudo":"alice","ville":"Lyon"}'
req POST /api/users '{"pseudo":"bob","ville":"Lyon"}'
req POST /api/users '{"pseudo":"carol"}'

# Cas 2 : pseudo vide
echo "[CAS 2] Créer un utilisateur avec pseudo vide → 400"
req POST /api/users '{"pseudo":"  "}'

# bob déclare sa compétence puis publie ses services
req PUT /api/users/2/skills '[{"nom":"Cuisine","niveau":"expert"}]' 2
req POST /api/services '{"titre":"Cours de cuisine","categorie":"Cuisine","duree_minutes":90,"credits":3}' 2
req POST /api/services '{"titre":"Banquet gastronomique","categorie":"Cuisine","duree_minutes":600,"credits":100}' 2
req POST /api/services '{"titre":"Atelier pâtisserie","categorie":"Cuisine","duree_minutes":120,"credits":4}' 2

titre "RÈGLES DE PUBLICATION ET DE DEMANDE"
# Cas 3 : publier sans avoir la compétence
echo "[CAS 3] Publier un service dans une catégorie sans compétence → 400"
req POST /api/services '{"titre":"Réparation PC","categorie":"Informatique","duree_minutes":30,"credits":1}' 1

# Cas 4 : échange sur son propre service
echo "[CAS 4] Demander un échange sur son propre service → 400"
req POST /api/exchanges '{"service_id":1}' 2

# Cas 5 : crédits insuffisants (service à 100, alice n'a que 10)
echo "[CAS 5] Demander un échange sans crédits suffisants → 400"
req POST /api/exchanges '{"service_id":2}' 1

# Cas 6 : demande valide puis conflit de réservation
echo "[CAS 6] Demander un échange (201) puis sur un service déjà réservé → 409"
req POST /api/exchanges '{"service_id":1}' 1
req POST /api/exchanges '{"service_id":1}' 3

titre "CYCLE DE VIE ET CRÉDITS"
solde alice 1; solde bob 2
# Cas 7 : acceptation → crédits bloqués
echo "[CAS 7] Accepter un échange → statut accepted, crédits bloqués"
req PUT /api/exchanges/1/accept '' 2
solde alice 1; solde bob 2

# Cas 8 : complétion → crédits transférés
echo "[CAS 8] Compléter un échange → statut completed, crédits transférés"
req PUT /api/exchanges/1/complete '' 1
solde alice 1; solde bob 2

# Cas 9 : annulation → crédits restitués
echo "[CAS 9] Annuler un échange → crédits restitués"
req POST /api/exchanges '{"service_id":3}' 3
req PUT /api/exchanges/2/accept '' 2
solde carol 3
req PUT /api/exchanges/2/cancel '' 3
solde carol 3

titre "ÉVALUATIONS"
# Cas 10 : noter un échange non terminé
echo "[CAS 10] Noter un échange non terminé → 400"
req POST /api/exchanges '{"service_id":3}' 1
req POST /api/exchanges/3/review '{"note":5}' 1

# Cas 11 : noter deux fois le même échange
echo "[CAS 11] Noter un échange terminé (201) puis une seconde fois → 400"
req POST /api/exchanges/1/review '{"note":5,"commentaire":"Parfait"}' 1
req POST /api/exchanges/1/review '{"note":3}' 1

titre "TABLEAU DE BORD"
# Cas 12 : statistiques cohérentes
echo "[CAS 12] Récupérer les stats → valeurs cohérentes (solde = gagné - dépensé)"
req GET /api/users/2/stats
req GET /api/users/1/stats

titre "FIN DE LA DÉMONSTRATION"
