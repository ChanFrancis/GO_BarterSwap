package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// insertReview enregistre un avis sur un échange terminé. L'auteur doit être
// une partie de l'échange ; la cible est l'autre partie. Un seul avis par
// auteur et par échange (contrainte d'unicité en base).
func (a *app) insertReview(ctx context.Context, exchangeID, authorID, note int, commentaire string) (Review, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Review{}, err
	}
	defer tx.Rollback()

	var requesterID, ownerID int
	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT requester_id, owner_id, status FROM exchanges WHERE id = $1`, exchangeID).
		Scan(&requesterID, &ownerID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return Review{}, ErrIntrouvable
	}
	if err != nil {
		return Review{}, err
	}

	if authorID != requesterID && authorID != ownerID {
		return Review{}, ErrInterdit
	}
	if status != StatusCompleted {
		return Review{}, ErrEchangeNonTermine
	}

	targetID := ownerID
	if authorID == ownerID {
		targetID = requesterID
	}

	var rv Review
	err = tx.QueryRowContext(ctx,
		`INSERT INTO reviews (exchange_id, author_id, target_id, note, commentaire)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, exchange_id, author_id, target_id, note, commentaire, created_at`,
		exchangeID, authorID, targetID, note, commentaire).
		Scan(&rv.ID, &rv.ExchangeID, &rv.AuthorID, &rv.TargetID, &rv.Note, &rv.Commentaire, &rv.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "reviews_exchange_id_author_id_key") {
			return Review{}, ErrDejaNote
		}
		return Review{}, err
	}
	return rv, tx.Commit()
}

// reviewsForUser retourne les avis reçus par un utilisateur.
func (a *app) reviewsForUser(ctx context.Context, targetID int) ([]Review, error) {
	return a.scanReviews(ctx,
		`SELECT id, exchange_id, author_id, target_id, note, commentaire, created_at
		 FROM reviews WHERE target_id = $1 ORDER BY created_at DESC`, targetID)
}

// reviewsForService retourne les avis portant sur les échanges d'un service.
func (a *app) reviewsForService(ctx context.Context, serviceID int) ([]Review, error) {
	return a.scanReviews(ctx,
		`SELECT r.id, r.exchange_id, r.author_id, r.target_id, r.note, r.commentaire, r.created_at
		 FROM reviews r JOIN exchanges e ON e.id = r.exchange_id
		 WHERE e.service_id = $1 ORDER BY r.created_at DESC`, serviceID)
}

func (a *app) scanReviews(ctx context.Context, query string, args ...any) ([]Review, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []Review{}
	for rows.Next() {
		var rv Review
		if err := rows.Scan(&rv.ID, &rv.ExchangeID, &rv.AuthorID, &rv.TargetID,
			&rv.Note, &rv.Commentaire, &rv.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

// fetchUserStats agrège les statistiques d'un utilisateur.
func (a *app) fetchUserStats(ctx context.Context, userID int) (UserStats, error) {
	if err := a.userExists(ctx, userID); err != nil {
		return UserStats{}, err
	}

	s := UserStats{UserID: userID}
	if err := a.db.QueryRowContext(ctx,
		`SELECT count(*) FROM services WHERE provider_id = $1 AND actif = true`,
		userID).Scan(&s.ServicesActifs); err != nil {
		return UserStats{}, err
	}
	if err := a.db.QueryRowContext(ctx,
		`SELECT count(*) FROM exchanges
		 WHERE (requester_id = $1 OR owner_id = $1) AND status = $2`,
		userID, StatusCompleted).Scan(&s.EchangesCompletes); err != nil {
		return UserStats{}, err
	}
	// Solde et totaux dérivés du journal : gagné = crédits entrants,
	// dépensé = crédits sortants, solde = gagné - dépensé.
	if err := a.db.QueryRowContext(ctx,
		`SELECT
		   COALESCE(SUM(montant), 0),
		   COALESCE(SUM(montant) FILTER (WHERE montant > 0), 0),
		   COALESCE(-SUM(montant) FILTER (WHERE montant < 0), 0)
		 FROM credit_transactions WHERE user_id = $1`,
		userID).Scan(&s.CreditBalance, &s.TotalGagne, &s.TotalDepense); err != nil {
		return UserStats{}, err
	}
	if err := a.db.QueryRowContext(ctx,
		`SELECT count(*), COALESCE(AVG(note), 0) FROM reviews WHERE target_id = $1`,
		userID).Scan(&s.NbAvis, &s.NoteMoyenne); err != nil {
		return UserStats{}, err
	}
	return s, nil
}
