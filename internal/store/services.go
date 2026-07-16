package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

// ServiceFilter porte les filtres de recherche côté serveur.
type ServiceFilter struct {
	Categorie string
	Ville     string
	Search    string
}

// InsertService crée une annonce et retourne le service complet.
func (s *Store) InsertService(ctx context.Context, providerID int, in barterswap.ServiceInput) (barterswap.Service, error) {
	var sv barterswap.Service
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO services (provider_id, titre, description, categorie, duree_minutes, credits, ville)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at`,
		providerID, in.Titre, in.Description, in.Categorie, in.DureeMinutes, in.Credits, in.Ville).
		Scan(&sv.ID, &sv.ProviderID, &sv.Titre, &sv.Description, &sv.Categorie,
			&sv.DureeMinutes, &sv.Credits, &sv.Ville, &sv.Actif, &sv.CreatedAt)
	return sv, err
}

// FetchService retourne une annonce par son identifiant.
func (s *Store) FetchService(ctx context.Context, id int) (barterswap.Service, error) {
	var sv barterswap.Service
	err := s.db.QueryRowContext(ctx,
		`SELECT id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at
		 FROM services WHERE id = $1`, id).
		Scan(&sv.ID, &sv.ProviderID, &sv.Titre, &sv.Description, &sv.Categorie,
			&sv.DureeMinutes, &sv.Credits, &sv.Ville, &sv.Actif, &sv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return sv, barterswap.ErrIntrouvable
	}
	return sv, err
}

// UpdateService modifie une annonce ; seul le propriétaire y est autorisé.
func (s *Store) UpdateService(ctx context.Context, id, callerID int, in barterswap.ServiceInput) (barterswap.Service, error) {
	if err := s.ensureServiceOwner(ctx, id, callerID); err != nil {
		return barterswap.Service{}, err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE services SET titre = $1, description = $2, categorie = $3,
		 duree_minutes = $4, credits = $5, ville = $6 WHERE id = $7`,
		in.Titre, in.Description, in.Categorie, in.DureeMinutes, in.Credits, in.Ville, id)
	if err != nil {
		return barterswap.Service{}, err
	}
	return s.FetchService(ctx, id)
}

// DeleteService supprime une annonce ; seul le propriétaire y est autorisé.
func (s *Store) DeleteService(ctx context.Context, id, callerID int) error {
	if err := s.ensureServiceOwner(ctx, id, callerID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM services WHERE id = $1`, id)
	return err
}

// ensureServiceOwner renvoie ErrIntrouvable si l'annonce n'existe pas, ou
// ErrInterdit si l'appelant n'en est pas le propriétaire.
func (s *Store) ensureServiceOwner(ctx context.Context, id, callerID int) error {
	var providerID int
	err := s.db.QueryRowContext(ctx,
		`SELECT provider_id FROM services WHERE id = $1`, id).Scan(&providerID)
	if errors.Is(err, sql.ErrNoRows) {
		return barterswap.ErrIntrouvable
	}
	if err != nil {
		return err
	}
	if providerID != callerID {
		return barterswap.ErrInterdit
	}
	return nil
}

// ListServices retourne les annonces actives correspondant aux filtres.
func (s *Store) ListServices(ctx context.Context, f ServiceFilter) ([]barterswap.Service, error) {
	query := `SELECT id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at
	          FROM services WHERE actif = true`
	args := []any{}
	if f.Categorie != "" {
		args = append(args, f.Categorie)
		query += fmt.Sprintf(" AND categorie = $%d", len(args))
	}
	if f.Ville != "" {
		args = append(args, f.Ville)
		query += fmt.Sprintf(" AND ville ILIKE $%d", len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		query += fmt.Sprintf(" AND (titre ILIKE $%d OR description ILIKE $%d)", len(args), len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	services := []barterswap.Service{}
	for rows.Next() {
		var sv barterswap.Service
		if err := rows.Scan(&sv.ID, &sv.ProviderID, &sv.Titre, &sv.Description, &sv.Categorie,
			&sv.DureeMinutes, &sv.Credits, &sv.Ville, &sv.Actif, &sv.CreatedAt); err != nil {
			return nil, err
		}
		services = append(services, sv)
	}
	return services, rows.Err()
}
