package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Accès base de données des échanges. Toutes les transitions passent par une
// transaction avec verrous de lignes : c'est la base, et non un mutex, qui
// sérialise les accès concurrents (contrainte du sujet).

// exchangeLock est un échange verrouillé, enrichi du coût du service.
type exchangeLock struct {
	Exchange
	credits int
}

// createExchange crée une demande d'échange après vérification des règles
// métier : service existant, pas le sien, aucun échange en cours, crédits
// suffisants.
func (a *app) createExchange(ctx context.Context, requesterID, serviceID int) (Exchange, error) {
	if err := a.userExists(ctx, requesterID); err != nil {
		return Exchange{}, err
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Exchange{}, err
	}
	defer tx.Rollback()

	var providerID, credits int
	err = tx.QueryRowContext(ctx,
		`SELECT provider_id, credits FROM services WHERE id = $1 FOR UPDATE`, serviceID).
		Scan(&providerID, &credits)
	if errors.Is(err, sql.ErrNoRows) {
		return Exchange{}, ErrIntrouvable
	}
	if err != nil {
		return Exchange{}, err
	}
	if providerID == requesterID {
		return Exchange{}, ErrServicePropre
	}

	// Un service ne peut avoir qu'un seul échange pending ou accepted.
	var enCours bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM exchanges
		 WHERE service_id = $1 AND status IN ($2, $3))`,
		serviceID, StatusPending, StatusAccepted).Scan(&enCours)
	if err != nil {
		return Exchange{}, err
	}
	if enCours {
		return Exchange{}, ErrDejaReserve
	}

	solde, err := balance(ctx, tx, requesterID)
	if err != nil {
		return Exchange{}, err
	}
	if solde < credits {
		return Exchange{}, ErrCreditsInsuffisants
	}

	var ex Exchange
	err = tx.QueryRowContext(ctx,
		`INSERT INTO exchanges (service_id, requester_id, owner_id) VALUES ($1, $2, $3)
		 RETURNING id, service_id, requester_id, owner_id, status, created_at, updated_at`,
		serviceID, requesterID, providerID).
		Scan(&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID, &ex.Status, &ex.CreatedAt, &ex.UpdatedAt)
	if err != nil {
		return Exchange{}, err
	}
	return ex, tx.Commit()
}

// acceptExchange (par l'offreur) bloque les crédits du demandeur : ils sont
// débités mais pas encore crédités à l'offreur.
func (a *app) acceptExchange(ctx context.Context, id, callerID int) (Exchange, error) {
	return a.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID {
			return ErrInterdit
		}
		if e.Status != StatusPending {
			return ErrTransitionInvalide
		}
		// Verrou du demandeur pour sérialiser les vérifications de solde.
		if _, err := tx.ExecContext(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, e.RequesterID); err != nil {
			return err
		}
		solde, err := balance(ctx, tx, e.RequesterID)
		if err != nil {
			return err
		}
		if solde < e.credits {
			return ErrCreditsInsuffisants
		}
		if err := setExchangeStatus(ctx, tx, id, StatusAccepted); err != nil {
			return err
		}
		return addTransaction(ctx, tx, e.RequesterID, id, -e.credits, "spend")
	})
}

// completeExchange transfère définitivement les crédits à l'offreur.
func (a *app) completeExchange(ctx context.Context, id, callerID int) (Exchange, error) {
	return a.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID && callerID != e.RequesterID {
			return ErrInterdit
		}
		if e.Status != StatusAccepted {
			return ErrTransitionInvalide
		}
		if err := setExchangeStatus(ctx, tx, id, StatusCompleted); err != nil {
			return err
		}
		return addTransaction(ctx, tx, e.OwnerID, id, e.credits, "earn")
	})
}

// cancelExchange (demandeur ou offreur) restitue les crédits bloqués.
func (a *app) cancelExchange(ctx context.Context, id, callerID int) (Exchange, error) {
	return a.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID && callerID != e.RequesterID {
			return ErrInterdit
		}
		if e.Status != StatusAccepted {
			return ErrTransitionInvalide
		}
		if err := setExchangeStatus(ctx, tx, id, StatusCancelled); err != nil {
			return err
		}
		return addTransaction(ctx, tx, e.RequesterID, id, e.credits, "refund")
	})
}

// rejectExchange (par l'offreur) refuse une demande en attente. Aucun crédit
// n'est bloqué à ce stade, il n'y a donc rien à restituer.
func (a *app) rejectExchange(ctx context.Context, id, callerID int) (Exchange, error) {
	return a.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID {
			return ErrInterdit
		}
		if e.Status != StatusPending {
			return ErrTransitionInvalide
		}
		return setExchangeStatus(ctx, tx, id, StatusRejected)
	})
}

// transition charge et verrouille l'échange, applique la règle fournie, puis
// valide la transaction et retourne l'échange à jour.
func (a *app) transition(ctx context.Context, id int, apply func(*sql.Tx, exchangeLock) error) (Exchange, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Exchange{}, err
	}
	defer tx.Rollback()

	var e exchangeLock
	err = tx.QueryRowContext(ctx,
		`SELECT e.id, e.service_id, e.requester_id, e.owner_id, e.status,
		        e.created_at, e.updated_at, s.credits
		 FROM exchanges e JOIN services s ON s.id = e.service_id
		 WHERE e.id = $1 FOR UPDATE OF e`, id).
		Scan(&e.ID, &e.ServiceID, &e.RequesterID, &e.OwnerID, &e.Status,
			&e.CreatedAt, &e.UpdatedAt, &e.credits)
	if errors.Is(err, sql.ErrNoRows) {
		return Exchange{}, ErrIntrouvable
	}
	if err != nil {
		return Exchange{}, err
	}

	if err := apply(tx, e); err != nil {
		return Exchange{}, err
	}
	if err := tx.Commit(); err != nil {
		return Exchange{}, err
	}
	return a.fetchExchange(ctx, id)
}

func setExchangeStatus(ctx context.Context, tx *sql.Tx, id int, status string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE exchanges SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	return err
}

func addTransaction(ctx context.Context, tx *sql.Tx, userID, exchangeID, montant int, typ string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
		 VALUES ($1, $2, $3, $4)`, userID, exchangeID, montant, typ)
	return err
}

// fetchExchange retourne un échange par son identifiant.
func (a *app) fetchExchange(ctx context.Context, id int) (Exchange, error) {
	var ex Exchange
	err := a.db.QueryRowContext(ctx,
		`SELECT id, service_id, requester_id, owner_id, status, created_at, updated_at
		 FROM exchanges WHERE id = $1`, id).
		Scan(&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID, &ex.Status, &ex.CreatedAt, &ex.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ex, ErrIntrouvable
	}
	return ex, err
}

// listExchanges retourne les échanges impliquant l'utilisateur (demandés ou
// reçus), éventuellement filtrés par statut.
func (a *app) listExchanges(ctx context.Context, userID int, status string) ([]Exchange, error) {
	query := `SELECT id, service_id, requester_id, owner_id, status, created_at, updated_at
	          FROM exchanges WHERE (requester_id = $1 OR owner_id = $1)`
	args := []any{userID}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	exchanges := []Exchange{}
	for rows.Next() {
		var ex Exchange
		if err := rows.Scan(&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID,
			&ex.Status, &ex.CreatedAt, &ex.UpdatedAt); err != nil {
			return nil, err
		}
		exchanges = append(exchanges, ex)
	}
	return exchanges, rows.Err()
}
