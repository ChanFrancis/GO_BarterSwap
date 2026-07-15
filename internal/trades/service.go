// Package trades implémente les offres de troc : un utilisateur propose un
// ou plusieurs de ses objets contre l'objet d'un autre. À l'acceptation, la
// propriété des objets est échangée.
package trades

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ChanFrancis/GO_BarterSwap/internal/items"
)

// Statuts d'une offre de troc.
const (
	StatusPending   = "en_attente"
	StatusAccepted  = "acceptée"
	StatusDeclined  = "refusée"
	StatusCancelled = "annulée"
)

var (
	ErrNotFound      = errors.New("offre introuvable")
	ErrForbidden     = errors.New("vous n'êtes pas concerné par cette offre")
	ErrNotPending    = errors.New("cette offre a déjà été traitée")
	ErrOwnItem       = errors.New("impossible de faire une offre sur son propre objet")
	ErrItemsRequired = errors.New("une offre doit proposer au moins un objet")
	ErrUnavailable   = errors.New("un des objets n'est plus disponible")
	ErrNotYourItems  = errors.New("vous ne pouvez proposer que vos propres objets disponibles")
)

type Offer struct {
	ID              int64      `json:"id"`
	ProposerID      int64      `json:"proposer_id"`
	RequestedItemID int64      `json:"requested_item_id"`
	OfferedItemIDs  []int64    `json:"offered_item_ids"`
	Message         string     `json:"message"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	DecidedAt       *time.Time `json:"decided_at,omitempty"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Create propose un troc : les objets offerts doivent appartenir au
// proposeur et être disponibles, l'objet demandé doit appartenir à
// quelqu'un d'autre.
func (s *Service) Create(ctx context.Context, proposerID, requestedItemID int64, offeredItemIDs []int64, message string) (*Offer, error) {
	if len(offeredItemIDs) == 0 {
		return nil, ErrItemsRequired
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var requestedOwnerID int64
	var requestedStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT owner_id, status FROM items WHERE id = $1`, requestedItemID).
		Scan(&requestedOwnerID, &requestedStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, items.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if requestedOwnerID == proposerID {
		return nil, ErrOwnItem
	}
	if requestedStatus != items.StatusAvailable {
		return nil, ErrUnavailable
	}

	// Tous les objets offerts doivent appartenir au proposeur et être
	// disponibles.
	var count int
	err = tx.QueryRowContext(ctx,
		`SELECT count(*) FROM items WHERE id = ANY($1) AND owner_id = $2 AND status = $3`,
		offeredItemIDs, proposerID, items.StatusAvailable).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count != len(offeredItemIDs) {
		return nil, ErrNotYourItems
	}

	offer := &Offer{ProposerID: proposerID, RequestedItemID: requestedItemID,
		OfferedItemIDs: offeredItemIDs, Message: message}
	err = tx.QueryRowContext(ctx,
		`INSERT INTO trade_offers (proposer_id, requested_item_id, message)
		 VALUES ($1, $2, $3) RETURNING id, status, created_at`,
		proposerID, requestedItemID, message).
		Scan(&offer.ID, &offer.Status, &offer.CreatedAt)
	if err != nil {
		return nil, err
	}
	for _, itemID := range offeredItemIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO trade_offer_items (offer_id, item_id) VALUES ($1, $2)`,
			offer.ID, itemID); err != nil {
			return nil, err
		}
	}
	return offer, tx.Commit()
}

// Accept finalise le troc : seul le propriétaire de l'objet demandé peut
// accepter. La propriété des objets est échangée et les autres offres en
// attente sur ces objets sont refusées.
func (s *Service) Accept(ctx context.Context, userID, offerID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var offer struct {
		proposerID, requestedItemID, ownerID int64
		status, itemStatus                   string
	}
	err = tx.QueryRowContext(ctx,
		`SELECT t.proposer_id, t.requested_item_id, t.status, i.owner_id, i.status
		 FROM trade_offers t JOIN items i ON i.id = t.requested_item_id
		 WHERE t.id = $1 FOR UPDATE OF t, i`, offerID).
		Scan(&offer.proposerID, &offer.requestedItemID, &offer.status,
			&offer.ownerID, &offer.itemStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if offer.ownerID != userID {
		return ErrForbidden
	}
	if offer.status != StatusPending {
		return ErrNotPending
	}
	if offer.itemStatus != items.StatusAvailable {
		return ErrUnavailable
	}

	offeredItemIDs, err := s.offeredItems(ctx, tx, offerID)
	if err != nil {
		return err
	}

	// Les objets offerts doivent toujours appartenir au proposeur et être
	// disponibles au moment de l'acceptation. FOR UPDATE les verrouille
	// contre une acceptation concurrente.
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM items WHERE id = ANY($1) AND owner_id = $2 AND status = $3 FOR UPDATE`,
		offeredItemIDs, offer.proposerID, items.StatusAvailable)
	if err != nil {
		return err
	}
	lockedCount := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		lockedCount++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if lockedCount != len(offeredItemIDs) {
		return ErrUnavailable
	}

	// Échange de propriété : l'objet demandé va au proposeur, les objets
	// offerts vont à l'accepteur.
	if _, err := tx.ExecContext(ctx,
		`UPDATE items SET owner_id = $1, updated_at = now() WHERE id = $2`,
		offer.proposerID, offer.requestedItemID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE items SET owner_id = $1, updated_at = now() WHERE id = ANY($2)`,
		userID, offeredItemIDs); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE trade_offers SET status = $1, decided_at = now() WHERE id = $2`,
		StatusAccepted, offerID); err != nil {
		return err
	}

	// Les autres offres en attente impliquant ces objets n'ont plus de sens.
	allItemIDs := append(offeredItemIDs, offer.requestedItemID)
	if _, err := tx.ExecContext(ctx,
		`UPDATE trade_offers SET status = $1, decided_at = now()
		 WHERE status = $2 AND id <> $3
		   AND (requested_item_id = ANY($4)
		        OR id IN (SELECT offer_id FROM trade_offer_items WHERE item_id = ANY($4)))`,
		StatusDeclined, StatusPending, offerID, allItemIDs); err != nil {
		return err
	}

	return tx.Commit()
}

// Decline refuse une offre reçue ; Cancel annule une offre envoyée.
func (s *Service) Decline(ctx context.Context, userID, offerID int64) error {
	return s.close(ctx, userID, offerID, StatusDeclined, false)
}

func (s *Service) Cancel(ctx context.Context, userID, offerID int64) error {
	return s.close(ctx, userID, offerID, StatusCancelled, true)
}

func (s *Service) close(ctx context.Context, userID, offerID int64, newStatus string, byProposer bool) error {
	var proposerID, ownerID int64
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT t.proposer_id, t.status, i.owner_id
		 FROM trade_offers t JOIN items i ON i.id = t.requested_item_id
		 WHERE t.id = $1`, offerID).Scan(&proposerID, &status, &ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	allowed := ownerID
	if byProposer {
		allowed = proposerID
	}
	if userID != allowed {
		return ErrForbidden
	}
	if status != StatusPending {
		return ErrNotPending
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE trade_offers SET status = $1, decided_at = now() WHERE id = $2 AND status = $3`,
		newStatus, offerID, StatusPending)
	return err
}

// ListForUser retourne les offres envoyées et reçues par l'utilisateur.
func (s *Service) ListForUser(ctx context.Context, userID int64) (sent, received []Offer, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT t.id, t.proposer_id, t.requested_item_id, t.message, t.status,
		        t.created_at, t.decided_at, i.owner_id
		 FROM trade_offers t JOIN items i ON i.id = t.requested_item_id
		 WHERE t.proposer_id = $1 OR i.owner_id = $1
		 ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	sent, received = []Offer{}, []Offer{}
	for rows.Next() {
		var o Offer
		var ownerID int64
		if err := rows.Scan(&o.ID, &o.ProposerID, &o.RequestedItemID, &o.Message,
			&o.Status, &o.CreatedAt, &o.DecidedAt, &ownerID); err != nil {
			return nil, nil, err
		}
		if o.OfferedItemIDs, err = s.offeredItems(ctx, s.db, o.ID); err != nil {
			return nil, nil, err
		}
		if o.ProposerID == userID {
			sent = append(sent, o)
		} else {
			received = append(received, o)
		}
	}
	return sent, received, rows.Err()
}

// querier permet d'exécuter la même requête sur *sql.DB ou *sql.Tx.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *Service) offeredItems(ctx context.Context, q querier, offerID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT item_id FROM trade_offer_items WHERE offer_id = $1 ORDER BY item_id`, offerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
