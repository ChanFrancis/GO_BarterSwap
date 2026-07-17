package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

// exchangeLock est un échange verrouillé, enrichi du coût du service.
type exchangeLock struct {
	barterswap.Exchange
	credits int
}

// CreateExchange crée une demande d'échange après vérification des règles
// métier : service existant, pas le sien, aucun échange en cours, crédits
// suffisants.
func (s *Store) CreateExchange(ctx context.Context, requesterID, serviceID int) (barterswap.Exchange, error) {
	if err := s.UserExists(ctx, requesterID); err != nil {
		return barterswap.Exchange{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return barterswap.Exchange{}, err
	}
	defer tx.Rollback()

	var providerID, credits int
	var actif bool
	err = tx.QueryRowContext(ctx,
		`SELECT provider_id, credits, actif FROM services WHERE id = $1 FOR UPDATE`, serviceID).
		Scan(&providerID, &credits, &actif)
	if errors.Is(err, sql.ErrNoRows) {
		return barterswap.Exchange{}, barterswap.ErrIntrouvable
	}
	if err != nil {
		return barterswap.Exchange{}, err
	}
	// Un service archivé (soft-delete) n'est plus réservable : il apparaît
	// comme introuvable pour les autres utilisateurs.
	if !actif {
		return barterswap.Exchange{}, barterswap.ErrIntrouvable
	}
	if providerID == requesterID {
		return barterswap.Exchange{}, barterswap.ErrServicePropre
	}

	// Un service ne peut avoir qu'un seul échange pending ou accepted.
	var enCours bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM exchanges WHERE service_id = $1 AND status IN ($2, $3))`,
		serviceID, barterswap.StatusPending, barterswap.StatusAccepted).Scan(&enCours)
	if err != nil {
		return barterswap.Exchange{}, err
	}
	if enCours {
		return barterswap.Exchange{}, barterswap.ErrDejaReserve
	}

	solde, err := balance(ctx, tx, requesterID)
	if err != nil {
		return barterswap.Exchange{}, err
	}
	if solde < credits {
		return barterswap.Exchange{}, barterswap.ErrCreditsInsuffisants
	}

	var ex barterswap.Exchange
	err = tx.QueryRowContext(ctx,
		`INSERT INTO exchanges (service_id, requester_id, owner_id) VALUES ($1, $2, $3)
		 RETURNING id, service_id, requester_id, owner_id, status, created_at, updated_at`,
		serviceID, requesterID, providerID).
		Scan(&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID, &ex.Status, &ex.CreatedAt, &ex.UpdatedAt)
	if err != nil {
		return barterswap.Exchange{}, err
	}
	return ex, tx.Commit()
}

// AcceptExchange (par l'offreur) bloque les crédits du demandeur : ils sont
// débités mais pas encore crédités à l'offreur.
func (s *Store) AcceptExchange(ctx context.Context, id, callerID int) (barterswap.Exchange, error) {
	return s.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID {
			return barterswap.ErrInterdit
		}
		if e.Status != barterswap.StatusPending {
			return barterswap.ErrTransitionInvalide
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
			return barterswap.ErrCreditsInsuffisants
		}
		if err := setExchangeStatus(ctx, tx, id, barterswap.StatusAccepted); err != nil {
			return err
		}
		return addTransaction(ctx, tx, e.RequesterID, id, -e.credits, "spend")
	})
}

// CompleteExchange transfère définitivement les crédits à l'offreur.
func (s *Store) CompleteExchange(ctx context.Context, id, callerID int) (barterswap.Exchange, error) {
	return s.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID && callerID != e.RequesterID {
			return barterswap.ErrInterdit
		}
		if e.Status != barterswap.StatusAccepted {
			return barterswap.ErrTransitionInvalide
		}
		if err := setExchangeStatus(ctx, tx, id, barterswap.StatusCompleted); err != nil {
			return err
		}
		return addTransaction(ctx, tx, e.OwnerID, id, e.credits, "earn")
	})
}

// CancelExchange (demandeur ou offreur) restitue les crédits bloqués.
func (s *Store) CancelExchange(ctx context.Context, id, callerID int) (barterswap.Exchange, error) {
	return s.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID && callerID != e.RequesterID {
			return barterswap.ErrInterdit
		}
		if e.Status != barterswap.StatusAccepted {
			return barterswap.ErrTransitionInvalide
		}
		if err := setExchangeStatus(ctx, tx, id, barterswap.StatusCancelled); err != nil {
			return err
		}
		return addTransaction(ctx, tx, e.RequesterID, id, e.credits, "refund")
	})
}

// RejectExchange (par l'offreur) refuse une demande en attente. Aucun crédit
// n'est bloqué à ce stade, il n'y a donc rien à restituer.
func (s *Store) RejectExchange(ctx context.Context, id, callerID int) (barterswap.Exchange, error) {
	return s.transition(ctx, id, func(tx *sql.Tx, e exchangeLock) error {
		if callerID != e.OwnerID {
			return barterswap.ErrInterdit
		}
		if e.Status != barterswap.StatusPending {
			return barterswap.ErrTransitionInvalide
		}
		return setExchangeStatus(ctx, tx, id, barterswap.StatusRejected)
	})
}

// transition charge et verrouille l'échange, applique la règle fournie, puis
// valide la transaction et retourne l'échange à jour.
func (s *Store) transition(ctx context.Context, id int, apply func(*sql.Tx, exchangeLock) error) (barterswap.Exchange, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return barterswap.Exchange{}, err
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
		return barterswap.Exchange{}, barterswap.ErrIntrouvable
	}
	if err != nil {
		return barterswap.Exchange{}, err
	}

	if err := apply(tx, e); err != nil {
		return barterswap.Exchange{}, err
	}
	if err := tx.Commit(); err != nil {
		return barterswap.Exchange{}, err
	}
	return s.FetchExchange(ctx, id)
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

// FetchExchange retourne un échange par son identifiant.
func (s *Store) FetchExchange(ctx context.Context, id int) (barterswap.Exchange, error) {
	var ex barterswap.Exchange
	err := s.db.QueryRowContext(ctx,
		`SELECT id, service_id, requester_id, owner_id, status, created_at, updated_at
		 FROM exchanges WHERE id = $1`, id).
		Scan(&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID, &ex.Status, &ex.CreatedAt, &ex.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ex, barterswap.ErrIntrouvable
	}
	return ex, err
}

// ListExchanges retourne les échanges impliquant l'utilisateur (demandés ou
// reçus), éventuellement filtrés par statut.
func (s *Store) ListExchanges(ctx context.Context, userID int, status string) ([]barterswap.Exchange, error) {
	query := `SELECT id, service_id, requester_id, owner_id, status, created_at, updated_at
	          FROM exchanges WHERE (requester_id = $1 OR owner_id = $1)`
	args := []any{userID}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	exchanges := []barterswap.Exchange{}
	for rows.Next() {
		var ex barterswap.Exchange
		if err := rows.Scan(&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID,
			&ex.Status, &ex.CreatedAt, &ex.UpdatedAt); err != nil {
			return nil, err
		}
		exchanges = append(exchanges, ex)
	}
	return exchanges, rows.Err()
}
