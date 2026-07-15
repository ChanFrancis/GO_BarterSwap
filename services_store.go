package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Accès base de données des annonces de services.

// insertService crée une annonce et retourne le service complet.
func (a *app) insertService(ctx context.Context, providerID int, in serviceInput) (Service, error) {
	var s Service
	err := a.db.QueryRowContext(ctx,
		`INSERT INTO services (provider_id, titre, description, categorie, duree_minutes, credits, ville)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at`,
		providerID, in.Titre, in.Description, in.Categorie, in.DureeMinutes, in.Credits, in.Ville).
		Scan(&s.ID, &s.ProviderID, &s.Titre, &s.Description, &s.Categorie,
			&s.DureeMinutes, &s.Credits, &s.Ville, &s.Actif, &s.CreatedAt)
	return s, err
}

// fetchService retourne une annonce par son identifiant.
func (a *app) fetchService(ctx context.Context, id int) (Service, error) {
	var s Service
	err := a.db.QueryRowContext(ctx,
		`SELECT id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at
		 FROM services WHERE id = $1`, id).
		Scan(&s.ID, &s.ProviderID, &s.Titre, &s.Description, &s.Categorie,
			&s.DureeMinutes, &s.Credits, &s.Ville, &s.Actif, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return s, ErrIntrouvable
	}
	return s, err
}

// updateService modifie une annonce ; seul le propriétaire y est autorisé.
func (a *app) updateService(ctx context.Context, id, callerID int, in serviceInput) (Service, error) {
	if err := a.ensureServiceOwner(ctx, id, callerID); err != nil {
		return Service{}, err
	}
	_, err := a.db.ExecContext(ctx,
		`UPDATE services SET titre = $1, description = $2, categorie = $3,
		 duree_minutes = $4, credits = $5, ville = $6 WHERE id = $7`,
		in.Titre, in.Description, in.Categorie, in.DureeMinutes, in.Credits, in.Ville, id)
	if err != nil {
		return Service{}, err
	}
	return a.fetchService(ctx, id)
}

// deleteService supprime une annonce ; seul le propriétaire y est autorisé.
func (a *app) deleteService(ctx context.Context, id, callerID int) error {
	if err := a.ensureServiceOwner(ctx, id, callerID); err != nil {
		return err
	}
	_, err := a.db.ExecContext(ctx, `DELETE FROM services WHERE id = $1`, id)
	return err
}

// ensureServiceOwner renvoie ErrIntrouvable si l'annonce n'existe pas, ou
// ErrInterdit si l'appelant n'en est pas le propriétaire.
func (a *app) ensureServiceOwner(ctx context.Context, id, callerID int) error {
	var providerID int
	err := a.db.QueryRowContext(ctx,
		`SELECT provider_id FROM services WHERE id = $1`, id).Scan(&providerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrIntrouvable
	}
	if err != nil {
		return err
	}
	if providerID != callerID {
		return ErrInterdit
	}
	return nil
}

// serviceFilter porte les filtres de recherche côté serveur.
type serviceFilter struct {
	Categorie string
	Ville     string
	Search    string
}

// listServices retourne les annonces actives correspondant aux filtres.
func (a *app) listServices(ctx context.Context, f serviceFilter) ([]Service, error) {
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

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	services := []Service{}
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.ProviderID, &s.Titre, &s.Description, &s.Categorie,
			&s.DureeMinutes, &s.Credits, &s.Ville, &s.Actif, &s.CreatedAt); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}
