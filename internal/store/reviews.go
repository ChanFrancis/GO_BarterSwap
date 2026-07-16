package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

// InsertReview enregistre un avis sur un échange terminé. L'auteur doit être
// une partie de l'échange ; la cible est l'autre partie. Un seul avis par
// auteur et par échange (contrainte d'unicité en base).
func (s *Store) InsertReview(ctx context.Context, exchangeID, authorID, note int, commentaire string) (barterswap.Review, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return barterswap.Review{}, err
	}
	defer tx.Rollback()

	var requesterID, ownerID int
	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT requester_id, owner_id, status FROM exchanges WHERE id = $1`, exchangeID).
		Scan(&requesterID, &ownerID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return barterswap.Review{}, barterswap.ErrIntrouvable
	}
	if err != nil {
		return barterswap.Review{}, err
	}

	if authorID != requesterID && authorID != ownerID {
		return barterswap.Review{}, barterswap.ErrInterdit
	}
	if status != barterswap.StatusCompleted {
		return barterswap.Review{}, barterswap.ErrEchangeNonTermine
	}

	targetID := ownerID
	if authorID == ownerID {
		targetID = requesterID
	}

	var rv barterswap.Review
	err = tx.QueryRowContext(ctx,
		`INSERT INTO reviews (exchange_id, author_id, target_id, note, commentaire)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, exchange_id, author_id, target_id, note, commentaire, created_at`,
		exchangeID, authorID, targetID, note, commentaire).
		Scan(&rv.ID, &rv.ExchangeID, &rv.AuthorID, &rv.TargetID, &rv.Note, &rv.Commentaire, &rv.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "reviews_exchange_id_author_id_key") {
			return barterswap.Review{}, barterswap.ErrDejaNote
		}
		return barterswap.Review{}, err
	}
	return rv, tx.Commit()
}

// ReviewsForUser retourne les avis reçus par un utilisateur.
func (s *Store) ReviewsForUser(ctx context.Context, targetID int) ([]barterswap.Review, error) {
	return s.scanReviews(ctx,
		`SELECT id, exchange_id, author_id, target_id, note, commentaire, created_at
		 FROM reviews WHERE target_id = $1 ORDER BY created_at DESC`, targetID)
}

// ReviewsForService retourne les avis portant sur les échanges d'un service.
func (s *Store) ReviewsForService(ctx context.Context, serviceID int) ([]barterswap.Review, error) {
	return s.scanReviews(ctx,
		`SELECT r.id, r.exchange_id, r.author_id, r.target_id, r.note, r.commentaire, r.created_at
		 FROM reviews r JOIN exchanges e ON e.id = r.exchange_id
		 WHERE e.service_id = $1 ORDER BY r.created_at DESC`, serviceID)
}

func (s *Store) scanReviews(ctx context.Context, query string, args ...any) ([]barterswap.Review, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []barterswap.Review{}
	for rows.Next() {
		var rv barterswap.Review
		if err := rows.Scan(&rv.ID, &rv.ExchangeID, &rv.AuthorID, &rv.TargetID,
			&rv.Note, &rv.Commentaire, &rv.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

// FetchUserStats agrège les statistiques d'un utilisateur.
func (s *Store) FetchUserStats(ctx context.Context, userID int) (barterswap.UserStats, error) {
	if err := s.UserExists(ctx, userID); err != nil {
		return barterswap.UserStats{}, err
	}

	st := barterswap.UserStats{UserID: userID}
	if err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM services WHERE provider_id = $1 AND actif = true`,
		userID).Scan(&st.ServicesActifs); err != nil {
		return barterswap.UserStats{}, err
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM exchanges
		 WHERE (requester_id = $1 OR owner_id = $1) AND status = $2`,
		userID, barterswap.StatusCompleted).Scan(&st.EchangesCompletes); err != nil {
		return barterswap.UserStats{}, err
	}
	// Solde et totaux dérivés du journal : gagné = crédits entrants,
	// dépensé = crédits sortants, solde = gagné - dépensé.
	if err := s.db.QueryRowContext(ctx,
		`SELECT
		   COALESCE(SUM(montant), 0),
		   COALESCE(SUM(montant) FILTER (WHERE montant > 0), 0),
		   COALESCE(-SUM(montant) FILTER (WHERE montant < 0), 0)
		 FROM credit_transactions WHERE user_id = $1`,
		userID).Scan(&st.CreditBalance, &st.TotalGagne, &st.TotalDepense); err != nil {
		return barterswap.UserStats{}, err
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT count(*), COALESCE(AVG(note), 0) FROM reviews WHERE target_id = $1`,
		userID).Scan(&st.NbAvis, &st.NoteMoyenne); err != nil {
		return barterswap.UserStats{}, err
	}
	return st, nil
}
